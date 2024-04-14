package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"unicode"
)

func main() {
	html := flag.Bool("html", false, "Print html")
	anki := flag.Bool("anki", false, "Print anki")
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
	} else if *anki {
		PrintAnkiCards(kn, co, 21, 8.0)
	} else {
		PrintPlaintextReport(kn, co, 200, 8.0)
	}
}

func PrintAnkiCards(kn Knowledge, co Corpus, count int, maxComplexity float64) {
	for n := 1; n <= count; n++ {
		sen, _, delta, _ := Best(kn, co, maxComplexity)
		kn.Learn(delta)
		words, translations, err := MakeTranslation(sen)
		if err != nil {
			log.Panic(err)
		}
		for i := range words {
			fmt.Println(FormatSentence(words, translations, i))
		}
	}
}

func FormatSentence(words []string, translations []string, i int) string {
	res := ""
	for j, word := range words {
		if j == i {
			res += "<b><u>"
		}
		res += word + " "
		if j == i {
			res += "</u></b>"
		}
	}
	res += ";"
	for j, word := range translations {
		if j == i {
			res += "<b><u>"
		}
		res += word + " "
		if j == i {
			res += "</u></b>"
		}
	}
	return res
}

var prompt = `
	You will receive a sentence in Hebrew.

	Translate it to Russian word-by-word.
	However, function words, phrasemes etc. should be joined together.
	Also, try to make the translation as coherent as possible.

	You must print the result precisely in the following format:
	     WORD1:::TRANSLATION;;;
	     WORD2:::TRANSLATION;;;
	One line corresponds to one fragment.
	The original fragment and its translation are separated by 3 colors.
	The fragments are separated by 3 semicolons.
	Example:
		אני:::Я;;;
		ממש אוהב:::очень люблю;;;
		לאכול:::есть;;;

	Details:
	1. If there are errors of any kind (formating, punctuation, semantic)
	   in the original sentence then modify the sentence to correct them.
	2. The translation should have proper punctuation and formatting.
`

func MakeTranslation(sentence string) ([]string, []string, error) {
	res, err := AskOpenAI(prompt, sentence)
	if err != nil {
		return nil, nil, err
	}
	lines := strings.Split(strings.TrimSpace(res), ";;;")
	var as []string
	var bs []string
	for _, line := range lines {
		line := strings.TrimSpace(line)
		if line == "" {
			continue
		}
		tokens := strings.Split(line, ":::")
		if len(tokens) != 2 {
			return nil, nil, fmt.Errorf("Bad line: %s", tokens)
		}
		as = append(as, tokens[0])
		bs = append(bs, tokens[1])
	}
	return as, bs, nil
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
		sen, _, delta, _ := Best(kn, co, maxComplexity)
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
		sen, _, delta, usefulness := Best(kn, co, maxComplexity)
		if n > count || usefulness <= 0.0001 {
			break
		}
		kn.Learn(delta)
		sc := Usefulness(kn, co)
		fmt.Printf("%9d %6d %10s %.1f%% %2s %s\n", n, int(Complexity(kn)), "", sc*100, "", sen)
		n++
	}
}

func Best(kn Knowledge, co Corpus, maxComplexity float64) (string, []string, Knowledge, float64) {
	next := co.Sentences()
	var rawSentenceBest string
	var sentenceBest []string
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
			sentenceBest = sentence
			knowledgeDeltaBest = knowledgeDelta
			usefulnessBest = usefulness
		}
	}
	return rawSentenceBest, sentenceBest, knowledgeDeltaBest, usefulnessBest
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

	for _, unparsedSentence := range sentences {
		sentence := Words(unparsedSentence)
		if len(sentence) == 0 {
			continue
		}
		rawSentence := MakeRawSentence(unparsedSentence)
		co.rawSentences = append(co.rawSentences, rawSentence)
		co.sentences = append(co.sentences, sentence)
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

func MakeRawSentence(unparsedSentence string) string {
	unparsedSentence = strings.ReplaceAll(unparsedSentence, "\r\n", " ")
	unparsedSentence = strings.ReplaceAll(unparsedSentence, "\n", " ")

	// This regex pattern matches a string that starts and ends with a letter, capturing it for extraction
	pattern := regexp.MustCompile(`(?i)^[^\p{L}]*([\p{L}].*?[\p{L}])[^\p{L}]*$`)
	matches := pattern.FindStringSubmatch(unparsedSentence)

	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func Clean(unparsedSentence string) string {
	sentence := unparsedSentence
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

func Words(cleanedSentence string) []string {
	cleanedSentence = Clean(cleanedSentence)
	res := []string{}
	for _, word := range strings.FieldsFunc(cleanedSentence, IsSeparator) {
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

func AskOpenAI(systemPrompt string, userPrompt string) (string, error) {
	fname := os.ExpandEnv("$HOME/.yeda-openai-api-key.txt")
	keyfile, err := os.Open(fname)
	if err != nil {
		return "", err
	}
	defer keyfile.Close()

	apiKeyBytes, err := io.ReadAll(keyfile)
	if err != nil {
		return "", err
	}
	apiKey := strings.TrimSpace(string(apiKeyBytes))

	// Create a new request
	data := OpenAIRequest{
		Model:    "gpt-4-turbo",
		Messages: []OpenAIMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}},
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	log.Println("Request:", string(payloadBytes))
	body := bytes.NewReader(payloadBytes)

	// Create an HTTP client and make the request
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", body)
	if err != nil {
		return "", err
	}

	// Set the content type and authorization headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read and output the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	log.Println("Response:", string(responseBody))
	var response OpenAIResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return "", err
	}
	return response.Choices[0].Message.Content, nil
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

// Define the structs that match the JSON structure
type OpenAIResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint"`
}

type Choice struct {
	Index        int              `json:"index"`
	Message      OpenAIMessage    `json:"message"`
	Logprobs     *json.RawMessage `json:"logprobs"` // Use *json.RawMessage for null or detailed data
	FinishReason string           `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
