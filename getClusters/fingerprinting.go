package getClusters



/**
* Function Kgram - the simplest overlap methods
* @parameter text - The string to turn into fingerprints
* @parameter windowSize - the size of the sliding window to use
* @parameter hashType - the type of hashing function to use
* @returns array of strings - the array of fingerprints
*/

func Kgram(text string, windowSize int, hashType string) []byte{
    i := 0
    for i + windowSize < len(text) {
        ComputeHash
    }
}
