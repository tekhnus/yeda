package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"strings"
	"unicode"
)

func main() {
	html := flag.Bool("html", false, "Print html")
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("Usage: yeda <filename>")
	}
	filename := flag.Arg(0)

	log.Println("Started loading the corpus")
	co, err := MakeCorpus(filename)
	log.Println("Finished loading the corpus")
	if err != nil {
		log.Fatal(err)
	}
	kn := Knowledge{}
	if *html {
		PrintHTMLCards(kn, co, 50, 8.0)
	} else {
		PrintPlaintextReport(kn, co, 200, 8.0)
	}
}

func PrintHTMLCards(kn Knowledge, co Corpus, count int, maxComplexity float64) {
	fmt.Println(`<!DOCTYPE html>
<html dir="auto">
<head>
<meta charset="UTF-8">
<style>
p {
  font-size: 22px;
  font-family: serif;
 padding-bottom: 32px;
}

.card {
 page-break-inside: avoid;
 margin-right: 16%;
 margin-left: 16%;
}
</style>
<title>Text Document</title>
</head>
<body>`)
	for n := 1; n <= count; n++ {
		sen, delta, _ := Best(kn, co, maxComplexity)
		kn.Learn(delta)
		fmt.Println(`<div class="card">`)
		fmt.Println(`<h4>`, n, `</h4>`)
		fmt.Println(`<p>`, sen, `</p>`)
		fmt.Println(`<hr>`)
		fmt.Println(`</div>`)
	}
	fmt.Println(`
			</body>
		</html>
		`)
}

func PrintPlaintextReport(kn Knowledge, co Corpus, count int, maxComplexity float64) {
	fmt.Printf("%d words in corpus\n", len(co.wordCount))
	fmt.Println()

	fmt.Printf("sentences  words  word_percentage    sentence\n")
	n := 1
	for {
		sen, delta, usefulness := Best(kn, co, maxComplexity)
		if n > count || usefulness <= 0.0001 {
			break
		}
		kn.Learn(delta)
		sc := Usefulness(kn, co)
		fmt.Printf("%9d %6d %10s %.1f%% %2s %s\n", n, int(Complexity(kn)), "", sc*100, "", sen)
		n++
	}
}

func Best(kn Knowledge, co Corpus, maxComplexity float64) (string, Knowledge, float64) {
	next := co.Sentences()
	var rawSentenceBest string
	var knowledgeDeltaBest Knowledge
	usefulnessBest := math.Inf(-1)
	for rawSentence, sentence := next(); sentence != nil; rawSentence, sentence = next() {
		knowledgeDelta := kn.Delta(sentence)
		usefulness := Usefulness(knowledgeDelta, co)
		complexity := Complexity(knowledgeDelta)
		if complexity > maxComplexity {
			continue
		}
		if usefulnessBest < usefulness {
			rawSentenceBest = rawSentence
			knowledgeDeltaBest = knowledgeDelta
			usefulnessBest = usefulness
		}
	}
	return rawSentenceBest, knowledgeDeltaBest, usefulnessBest
}

func Usefulness(kn Knowledge, co Corpus) float64 {
	var res float64
	for word := range kn.Words {
		res += float64(co.wordCount[word])
	}
	return res / float64(co.totalWords)
}

func Complexity(delta Knowledge) float64 {
	return float64(len(delta.Words))
}

type Knowledge struct {
	Words map[string]bool
}

func (kn Knowledge) Delta(csen []string) Knowledge {
	delta := make(map[string]bool)
	for _, word := range csen {
		if !kn.Words[word] {
			delta[word] = true
		}
	}
	return Knowledge{delta}
}

func (kn *Knowledge) Learn(delta Knowledge) {
	if kn.Words == nil {
		kn.Words = make(map[string]bool)
	}
	for word := range delta.Words {
		kn.Words[word] = true
	}
}

type Corpus struct {
	rawSentences []string
	sentences    [][]string
	wordCount    map[string]int
	totalWords   int
}

func (co Corpus) Sentences() func() (string, []string) {
	i := 0
	return func() (string, []string) {
		if i == len(co.sentences) {
			return "", nil
		}
		rawSentence := co.rawSentences[i]
		sentence := co.sentences[i]
		i++
		return rawSentence, sentence
	}
}

func MakeCorpus(filename string) (Corpus, error) {
	co := Corpus{}
	fname := os.ExpandEnv(filename)
	file, err := os.Open(fname)
	if err != nil {
		return co, err
	}
	defer file.Close()

	bt, err := io.ReadAll(file)
	if err != nil {
		return co, err
	}
	sentences := Sentences(string(bt))

	for _, rawSentence := range sentences {
		cleanSentence := Clean(rawSentence)
		if cleanSentence == "" {
			continue
		}
		sen := Words(cleanSentence)
		niceSentence := rawSentence
		niceSentence = MakeNice(niceSentence)
		co.rawSentences = append(co.rawSentences, niceSentence)
		co.sentences = append(co.sentences, sen)
	}
	co.wordCount = make(map[string]int)
	next := co.Sentences()
	for _, sen := next(); sen != nil; _, sen = next() {
		for _, word := range sen {
			co.wordCount[word] += 1
			co.totalWords += 1
		}
	}
	log.Println("Text size:", len(bt))
	log.Println("Sentences:", len(sentences))
	log.Println("Words:", co.totalWords)
	log.Println("Unique words:", len(co.wordCount))
	return co, nil
}

func MakeNice(rawSentence string) string {
	rawSentence = strings.ReplaceAll(rawSentence, "\r\n", " ")
	rawSentence = strings.ReplaceAll(rawSentence, "\n", " ")

	// This regex pattern matches a string that starts and ends with a letter, capturing it for extraction
	pattern := regexp.MustCompile(`(?i)^[^a-z]*([a-z].*?[a-z])[^a-z]*$`)
	matches := pattern.FindStringSubmatch(rawSentence)

	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func Clean(rawSentence string) string {
	sentence := rawSentence
	sentence = strings.ReplaceAll(sentence, `“`, "")
	sentence = strings.ReplaceAll(sentence, `”`, "")
	sentence = strings.ReplaceAll(sentence, `‘`, "")
	sentence = strings.ReplaceAll(sentence, `’`, "")
	sentence = strings.ReplaceAll(sentence, `"`, "")
	sentence = strings.ReplaceAll(sentence, "\r\n", " ")
	sentence = strings.ReplaceAll(sentence, "\n", " ")
	sentence = strings.TrimSpace(sentence)
	return sentence
}

func Sentences(text string) []string {
	res := []string{}
	sentenceEnd := regexp.MustCompile(`[.?!]+`)
	indices := sentenceEnd.FindAllStringIndex(text, -1)
	start := 0
	for _, span := range indices {
		sentence := text[start:span[1]]
		start = span[1]
		if sentence == "" {
			continue
		}
		res = append(res, sentence)
	}
	return res
}

func Words(sen string) []string {
	res := []string{}
	for _, word := range strings.FieldsFunc(sen, IsSeparator) {
		word = strings.ToLower(word)
		if word == "" {
			continue
		}
		res = append(res, word)
	}
	return res
}

func IsSeparator(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '’'
}
