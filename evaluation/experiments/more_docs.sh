num_docs=(3 5 7 9 11 13 15)

for num in "${num_docs[@]}"
do
	echo "${num}"
	./postCorr -input="real_datasets/meetings/${num}" -groundtruth="real_datasets/meetings/ocr" -proportion=0.1
done