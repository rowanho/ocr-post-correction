package correction

import (
	"postCorr/common"
	"postCorr/flags"
	"postCorr/readWrite"

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

func modifyText(primaryDocumentID string, text []rune) []rune {
	var groundText []rune
	if flags.Logging && flags.Groundtruth != "" {
		groundText, _ = readWrite.ReadRunes(path.Join(flags.Groundtruth, primaryDocumentID))
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
					before := levenshtein.ComputeDistance(groundText, append(newText[:endPoint-1], text[i:]...))
					after := levenshtein.ComputeDistance(groundText, append(newText[:endPoint], text[i+1:]...))
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
					before := levenshtein.ComputeDistance(groundText, append(newText[:endPoint], text[i:]...))
					after := levenshtein.ComputeDistance(groundText, append(newText[:endPoint], text[i+1:]...))
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
			if flags.Logging {
				for l := endPoint; l < endPoint+len(additionIndices[primaryDocumentID][i]); l++ {
					if flags.Groundtruth != "" {
						before := levenshtein.ComputeDistance(groundText, append(newText[:l-1], text[i+1:]...))
						after := levenshtein.ComputeDistance(groundText, append(newText[:l], text[i+1:]...))
						if before < after {
							insEdits[l] = "worse"
						} else if before == after {
							insEdits[l] = "same"
						} else {
							insEdits[l] = "better"
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
	}
	substitutionGraph[primaryDocumentID] = subEdits
	deletionGraph[primaryDocumentID] = delEdits
	insertionGraph[primaryDocumentID] = insEdits
	return newText
}

/**
* There needs to be a function here that takes in the alignment graph and produces clusters
* We can ideally produce 1 cluster per alignment, if it's too small, we can stop
* The max distance level is how far we want to traverse the neighbours of the master's neighbours
* High max distances can lead to worse time complexity
**/

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
			readWrite.PlaintextWrite(docID, documents[docMap[docID]].Text)
		}
	}

	if flags.Logging {
		readWrite.SerialiseVote(reuseGraph)
		readWrite.SerialiseStartEnds(oldStartEndGraph, "old")
		readWrite.SerialiseStartEnds(reuseStartEndGraph, "new")
		readWrite.SerialiseEdits(substitutionGraph, "sub")
		readWrite.SerialiseEdits(deletionGraph, "del")
		readWrite.SerialiseEdits(insertionGraph, "ins")
		readWrite.SerialiseMVote(newVoteLogs)
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
