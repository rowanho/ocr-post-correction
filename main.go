package main

import (
	"postCorr/readWrite"
	"postCorr/common"
	"postCorr/alignment"
	"postCorr/queries"
	"postCorr/correction"
	"postCorr/flags"
	
	"fmt"
	"flag"
	"time"
	"sync"
	"context"
)

func main() {
	dirName  := flag.String("dir","test_dataset","path to dataset")
	formatType := flag.String("format", common.Plaintext, "the dataset file format")
	alignmentTolerance := flag.Int("tolerance", 10, "Tolerance for distances between alignments to identify as similar" )
	fpType := flag.String("fp", common.MinhashFP, "Fingeprinting method")
	jaccardThreshold := flag.Float64("jaccard", 0.05, "Jaccard index threshold for similarity")
	flag.Parse()
	
	flags.DirName = *dirName
	flags.FormatType = *formatType
	flags.AlignmentTolerance = *alignmentTolerance
	flags.FpType = *fpType
	flags.JaccardThreshold = *jaccardThreshold
	execute()
}


/**
* Executes the main program pipeline
**/
func execute() {
	totalCorrections := 0
	queries.CreateAlignmentIndex(common.AlignmentIndex)
	queries.CreateFingerprintIndex(common.FpIndex)
	//queries.CreateLSHFingerprintIndex(common.FpLSHIndex, 5, 7, 512)
	time.Sleep(1 * time.Second)

	docIDList, docsErr := readWrite.TraverseAndIndexDocs()

	if docsErr != nil {
		fmt.Println("Error indexing documents %s", docsErr)
		return
	}
	
	numDocs := len(docIDList)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
	    count, err := queries.CountDocs(common.DocumentIndex) 
	    if count == numDocs  || ctx.Err() != nil || err != nil {
	        break
	    }
	}
		
	likelyMatchingDocs := getSimilarDocuments(docIDList)
	
	fmt.Println(likelyMatchingDocs)
	alignmentCount := alignAndIndex(likelyMatchingDocs)
	fmt.Println(likelyMatchingDocs)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
	    count, err := queries.CountDocs(common.AlignmentIndex) 
	    if count == alignmentCount || ctx.Err() != nil || err != nil {
	        break
	    }
	}

	alignmentAdjacencyList := getSimilarAlignments(docIDList)
	fmt.Println(alignmentAdjacencyList)
	totalCorrections += correction.ClusterAndCorrectAlignments(alignmentAdjacencyList, 1)
	fmt.Println("Number of corrections made: ", totalCorrections)
	queries.DeleteIndexes([]string{common.AlignmentIndex, common.FpIndex, common.DocumentIndex, common.MinHashIndex})
}

func getSimilarDocuments(docIDList []string) map[string]map[string]bool{
	likelyMatchingDocs := make(map[string]map[string]bool, 0)
	for _, docID := range docIDList {
		if (flags.FpType == common.ModFP) {
			similarDocIDs, _ := queries.GetSimilarFps(common.FpIndex, docID, docIDList, flags.JaccardThreshold)
			likelyMatchingDocs[docID] = similarDocIDs			
		} else if (flags.FpType == common.MinhashFP) {
			similarDocIDs, _ := queries.GetSimilarMinHashes(common.MinHashIndex, docID, docIDList)
			likelyMatchingDocs[docID] = similarDocIDs			
		}
	}
	return likelyMatchingDocs	
}

func alignAndIndex(likelyMatchingDocs map[string]map[string]bool) int {
	alignmentCount := 0
	var wg sync.WaitGroup
	
	wg.Add(len(likelyMatchingDocs))
	for primID, secIDs := range likelyMatchingDocs {
		go func(primID string, secIDs map[string]bool){
			defer wg.Done()
			primDoc, _ := queries.GetDocByID(common.DocumentIndex, primID)
			for secID, _:= range secIDs {
				if _, exists := likelyMatchingDocs[secID][primID]; exists {
						delete(likelyMatchingDocs[secID],primID)
				}
				secDoc, _ := queries.GetDocByID(common.DocumentIndex, secID)
				alignments := alignment.GetAlignments(1.0, 2.0, primDoc, secDoc, 1, 0.0)
				for _, al := range alignments {
					queries.IndexAlignment(common.AlignmentIndex, al)
				}
				alignmentCount += 1
			}
		}(primID, secIDs)
	}
	wg.Wait()	
	return alignmentCount
}

func getSimilarAlignments(docIDList []string) map[string][]string {
	
	alignmentAdjacencyList := make(map[string][]string, 0)
	// Loop through all alignments
	for _, docID := range docIDList {
		fmt.Println(docID)
		alignments,_ := queries.GetAlignmentsByPrimID(common.AlignmentIndex, docID)
		fmt.Println(len(alignments))
		for _, al := range alignments {
			matchingAlignmentIds, _ := queries.GetMatchingAlignments(common.AlignmentIndex, 
																al, 
																flags.AlignmentTolerance)
			connectedAlignmentIds, _ := queries.GetConnectedAlignments(common.AlignmentIndex,
																	   al,
																  	   flags.AlignmentTolerance)
			alignmentAdjacencyList[al.ID] = append(matchingAlignmentIds, connectedAlignmentIds...)
		}
	}
	return alignmentAdjacencyList
}


