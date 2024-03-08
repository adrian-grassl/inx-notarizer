package common

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func GetRequest(endPoint string, token string) (*http.Response, error) {
	httpRequest, err := http.NewRequest(http.MethodGet, endPoint, nil)
	if err != nil {
		return nil, err
	}

	httpRequest.Header.Set("Accept", "application/json")
	if len(token) > 0 {
		httpRequest.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func PostRequest(endPoint string, token string, payload any) (*http.Response, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	bodyReader := bytes.NewReader(jsonBody)
	httpRequest, err := http.NewRequest(http.MethodPost, endPoint, bodyReader)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	if len(token) > 0 {
		httpRequest.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}

	return res, nil
}
