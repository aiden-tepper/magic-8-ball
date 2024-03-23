package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func queryAPIAndDisplayResult(question string, doneLoading chan bool, doneShaking chan bool) {
	// Use a goroutine for querying the API to not block the main thread
	go func() {
		defer func() {
			doneLoading <- true
			doneShaking <- true
		}()

		result, err := QueryHuggingFace(question)
		if err != nil {
			alert(fmt.Sprint(err))
			return
		}

		displayResult(result)
	}()
}

func QueryHuggingFace(question string) (map[string]string, error) {
	apiKey := os.Getenv("HF_TOKEN")
	prompt := fmt.Sprintf(`Pretend you are a magic 8 ball. I will give you a question, and you will respond in the way a magic 8 ball would, but make it funny and clever. Here is your question: '%s'. Reply in this format: {\"answer\": answer, \"explanation\": explanation}, where 'answer' is the few word answer that would show up on the magic 8-ball itself, and explanation is a sentence or two of explanation, humorous quips, or highly analytical statements.`, question)
	input := fmt.Sprintf(`{"inputs": "%s", "options": {"use_cache": false}}`, prompt)
	payload := bytes.NewBuffer([]byte(input))

	req, err := http.NewRequest("POST", modelEndpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %s", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %s", err)
	}

	var responseObject []Response
	err = json.Unmarshal(body, &responseObject)
	if err != nil {
		return nil, fmt.Errorf("error parsing response body: %s", err)
	}

	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(responseObject[0].Answer, -1)
	if len(matches) == 0 {
		return map[string]string{"answer": "Ask again!", "explanation": "give the 8 ball one more shake."}, nil
	}

	var result map[string]string
	err = json.Unmarshal([]byte(matches[1][0]), &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON object: %s", err)
	}

	if len(responseObject) > 0 {
		return map[string]string{"answer": result["answer"], "explanation": result["explanation"]}, nil
	}
	return nil, nil
}

func displayResult(result map[string]string) {
	app.QueueUpdateDraw(func() {
		if result != nil {
			output := fmt.Sprintf("[::b]%s\n\n[::i]%s", result["answer"], result["explanation"])
			_, yOffset := analyzeMultilineString(output)
			_, _, _, height := outputView.GetInnerRect()
			// Ensuring the result is vertically centered by adjusting the padding
			outputView.SetText(strings.Repeat("\n", (height/2)-(yOffset/2)) + output)
		} else {
			outputView.SetText("Could not retrieve an answer. Please try again.")
		}
	})
}
