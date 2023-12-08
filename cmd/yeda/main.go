package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"
	"unicode"
)

func main() {
	log.Println("Started loading the corpus")
	co, err := MakeCorpus()
	log.Println("Finished loading the corpus")
	if err != nil {
		log.Fatal(err)
	}
	kn := Knowledge{}
	n := 1
	for {
		log.Println("Started selecting best sentence")
		sen, delta, u := Best(kn, co)
		log.Println("Finished selecting best sentence")
		if u <= 0 {
			break
		}
		kn.Learn(delta)
		sc := Usefulness(kn, co)
		fmt.Printf("%4d %4d %.2f%% %s\n", n, int(Complexity(kn)), sc*100, sen)
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
		if comp > 10 {
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

func MakeCorpus() (Corpus, error) {
	co := Corpus{}
	fname := os.ExpandEnv("$HOME/.yeda-corpus.txt")
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
	return co, nil
}

func Sentences(text string) []string {
	var res []string
	sentences := strings.FieldsFunc(text, IsSentenceEnd)
	for _, sentence := range sentences {
		res = append(res, strings.ReplaceAll(strings.TrimSpace(sentence), "\n", " "))
	}
	return res
}

func Words(sen string) []string {
	var res []string
	for _, w := range strings.FieldsFunc(sen, IsSeparator) {
		res = append(res, strings.ToLower(w))
	}
	return res
}

func IsSentenceEnd(c rune) bool {
	return c == '.' || c == '?' || c == '!' || c == ';' || c == '”'
}

func IsSeparator(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '’'
}
