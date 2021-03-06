package correction

import (
	"postcorr/common"
	"postcorr/flags"
	"postcorr/iohandler"

	"fmt"
	"path"

	"github.com/rowanho/levenshtein"
	"github.com/schollz/progressbar"
)

type alignMap = struct {
	Mapping             map[int]int
	PrimaryDocumentID   string
	SecondaryDocumentID string
	Start               int
	End                 int
}


func getCopies(a1 []rune, a2 []rune)  ([]rune, []rune){
	c1 := make([]rune, len(a1))
	copy(c1, a1)
	c2 := make([]rune, len(a2))
	copy(c2, a2)
	return c1, c2
}


func modifyText(primaryDocumentID string, text []rune) []rune {
	var groundText []rune
	if flags.Logging && flags.Groundtruth != "" {
		groundText, _ = iohandler.ReadRunes(path.Join(flags.Groundtruth, primaryDocumentID))
	}
	subEdits := make(map[int]string)
	delEdits := make(map[int]string)
	insEdits := make(map[int]string)
	newText := make([]rune, 0)
	endPoint := 0
	modified := false
	sub := true
	for i := 0; i < len(text); i++ {
		if _, exists := removeIndices[primaryDocumentID][i]; exists {
			modified = true
			sub = false
		} else if _, exists := editIndices[primaryDocumentID][i]; exists {
			modified = true
			sub = true
			newText = append(newText, editIndices[primaryDocumentID][i])
		} else {
			modified = false
			newText = append(newText, text[i])
		}
		endPoint = len(newText)
		if flags.Logging && modified {
			if sub {
				if flags.Groundtruth != "" {
					newTextCP, textCP := getCopies(newText, text)
					before := levenshtein.ComputeDistance(groundText, append(newTextCP[:endPoint-1], textCP[i:]...))
					newTextCP, textCP = getCopies(newText, text)
					after := levenshtein.ComputeDistance(groundText, append(newTextCP[:endPoint], textCP[i+1:]...))
					if before < after {
						subEdits[endPoint-1] = "worse"
					} else if before == after {
						subEdits[endPoint-1] = "same"
					} else {
						subEdits[endPoint-1] = "better"
					}
				} else {
					subEdits[endPoint-1] = "same"
				}
				if _, exists := newVoteLogs[primaryDocumentID][endPoint-1]; !exists {
					newVoteLogs[primaryDocumentID][endPoint-1] = common.Vote{
						EditDict:   map[string]int{},
						InsertDict: map[string]int{},
					}
				}
				for key, val := range mVoteLogs[primaryDocumentID][i].EditDict {
					newVoteLogs[primaryDocumentID][endPoint-1].EditDict[key] = val
				}
			} else {
				if flags.Groundtruth != "" {
					newTextCP, textCP := getCopies(newText, text)
					before := levenshtein.ComputeDistance(groundText, append(newTextCP[:endPoint], textCP[i:]...))
					newTextCP, textCP = getCopies(newText, text)
					after := levenshtein.ComputeDistance(groundText, append(newTextCP[:endPoint], textCP[i+1:]...))
					if before < after {
						delEdits[i] = "worse"
					} else if before == after {
						delEdits[i] = "same"
					} else {
						delEdits[i] = "better"
					}
				} else {
					delEdits[i] = "same"
				}
			}
		}

		if _, exists := additionIndices[primaryDocumentID][i]; exists {
			endPoint = len(newText)
			newText = append(newText, additionIndices[primaryDocumentID][i]...)
			l := len(additionIndices[primaryDocumentID][i])
			if flags.Logging {
				if flags.Groundtruth != "" {
					newTextCP, textCP := getCopies(newText, text)
					before := levenshtein.ComputeDistance(groundText, append(newTextCP[:endPoint - l], textCP[i+1:]...))
					newTextCP, textCP = getCopies(newText, text)
					after := levenshtein.ComputeDistance(groundText, append(newTextCP, textCP[i+1:]...))
					for j := endPoint; j < endPoint + l; j++ {
						if before < after {
							insEdits[j] = "worse"
						} else if before == after {
							insEdits[j] = "same"
						} else {
							insEdits[j] = "better"
						}
					}
				} else {
					insEdits[l] = "same"
				}
				if _, exists := newVoteLogs[primaryDocumentID][endPoint-1]; !exists {
					newVoteLogs[primaryDocumentID][endPoint-1] = common.Vote{
						EditDict:   map[string]int{},
						InsertDict: map[string]int{},
					}
				}
				for key, val := range mVoteLogs[primaryDocumentID][i].InsertDict {
					newVoteLogs[primaryDocumentID][endPoint-1].InsertDict[key] = val
				}
			}
		}
	}
	substitutionGraph[primaryDocumentID] = subEdits
	deletionGraph[primaryDocumentID] = delEdits
	insertionGraph[primaryDocumentID] = insEdits
	return newText
}


// Takes in the alignment graph and produces clusters
// We can ideally produce 1 cluster per alignment, if it's too small, we can stop
// The max distance level is how far we want to traverse the neighbours of the master's neighbours
// High max distances can lead to worse time complexity
func ClusterAndCorrectAlignments(clustersList [][]string, alignments map[string]common.Alignment, documents []common.Document, docMap map[string]int) (map[string]bool, int) {
	bar := progressbar.New(100)
	totalCorrections := 0
	correctedDocs := make(map[string]bool)
	// Loop through the cluster list
	c := 0
	prev := 0
	for _, cluster := range clustersList {
		// Attempt to correct the primary document of the cluster
		if len(cluster) > 1 {
			alignmentMaps := make([]alignMap, len(cluster))
			primaryDocumentID := alignments[cluster[0]].PrimaryDocumentID
			for i, alignmentId := range cluster {
				alignmentMaps[i] = getAlignmentMap(alignments[alignmentId])
			}
			noCorrections := MajorityVote(primaryDocumentID, alignmentMaps, documents, docMap)
			totalCorrections += noCorrections
			if noCorrections > 0 {
				correctedDocs[primaryDocumentID] = true
			}
		}
		c += 1
		prog := (c * 100) / len(clustersList)
		if prog > prev {
			bar.Add(1)
			prev = prog
		}
	}
	fmt.Println("")
	for primaryDocumentID := range correctedDocs {
		correctedDocText := modifyText(primaryDocumentID, documents[docMap[primaryDocumentID]].Text)
		documents[docMap[primaryDocumentID]].Text = correctedDocText
	}
	if flags.WriteOutput {
		for docID := range correctedDocs {
			iohandler.PlaintextWrite(docID, documents[docMap[docID]].Text)
		}
	}

	if flags.Logging {
		iohandler.SerialiseVote(reuseGraph)
		iohandler.SerialiseStartEnds(oldStartEndGraph, "old")
		iohandler.SerialiseStartEnds(reuseStartEndGraph, "new")
		iohandler.SerialiseEdits(substitutionGraph, "sub")
		iohandler.SerialiseEdits(deletionGraph, "del")
		iohandler.SerialiseEdits(insertionGraph, "ins")
		iohandler.SerialiseMVote(newVoteLogs)
		iohandler.SerialiseDirname()
	}
	if flags.UseLM {
		fmt.Printf("Prevented %d\n", prevCount)
	}
	return correctedDocs, totalCorrections
}

func getAlignmentMap(al common.Alignment) alignMap {
	m := map[int]int{}
	for i, ind := range al.PrimaryAl {
		m[ind] = al.SecondaryAl[i]
	}
	a := alignMap{
		Mapping:             m,
		PrimaryDocumentID:   al.PrimaryDocumentID,
		SecondaryDocumentID: al.SecondaryDocumentID,
		Start:               al.PrimaryStartIndex,
		End:                 al.PrimaryEndIndex,
	}
	return a
}
