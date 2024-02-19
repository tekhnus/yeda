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

	log.Println("Started loading the corpus")
	co, err := MakeCorpus(os.Args[1])
	log.Println("Finished loading the corpus")
	if err != nil {
		log.Fatal(err)
	}
	kn := Knowledge{}
	if *html {
		PrintHTMLCards(kn, co)
	} else {
		PrintPlaintextReport(kn, co)
	}
}

func PrintHTMLCards(kn Knowledge, co Corpus) {
	fmt.Println(`<!DOCTYPE html>
<html dir="rtl">
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
	for n := 1; n <= 33; n++ {
		sen, delta, _ := Best(kn, co)
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

func PrintPlaintextReport(kn Knowledge, co Corpus) {
	n := 1
	for {
		// log.Println("Started selecting best sentence")
		sen, delta, u := Best(kn, co)
		// log.Println("Finished selecting best sentence")
		if u <= 0.0001 {
			break
		}
		kn.Learn(delta)
		sc := Usefulness(kn, co)
		fmt.Printf("%4d %4d / %4d %.2f%% %s\n", n, int(Complexity(kn)), len(co.wordCount), sc*100, sen)
		n++
	}
}

func Best(kn Knowledge, co Corpus) (string, Knowledge, float64) {
	next := co.Sentences()
	var bestRsent string
	var bestDelta Knowledge
	bestVal := math.Inf(-1)
	for rsen, sen := next(); sen != nil; rsen, sen = next() {
		delta := kn.Delta(sen)
		u := Usefulness(delta, co)
		comp := Complexity(delta)
		if comp > 8 {
			continue
		}
		if bestVal < u {
			bestRsent = rsen
			bestDelta = delta
			bestVal = u
		}
	}
	return bestRsent, bestDelta, bestVal
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
	rawsentences []string
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
		rres := co.rawsentences[i]
		res := co.sentences[i]
		i++
		return rres, res
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

	for _, rawsentence := range sentences {
		sen := Words(rawsentence)
		co.rawsentences = append(co.rawsentences, rawsentence)
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

func Sentences(text string) []string {
	text = strings.ReplaceAll(text, `“`, "")
	text = strings.ReplaceAll(text, `”`, "")
	text = strings.ReplaceAll(text, `"`, "")
	res := []string{}
	sentenceEnd := regexp.MustCompile(`[.?!]+`)
	indices := sentenceEnd.FindAllStringIndex(text, -1)
	start := 0
	for _, span := range indices {
		sentence := text[start:span[1]]
		start = span[1]
		sentence = strings.ReplaceAll(sentence, "\r\n", " ")
		sentence = strings.ReplaceAll(sentence, "\n", " ")
		sentence = strings.TrimSpace(sentence)
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
