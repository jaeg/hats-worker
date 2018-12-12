package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

func httpGet(url string) map[string]interface{} {
	resp, err := http.Get(url)
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	return map[string]interface{}{"body": string(body), "status": resp.StatusCode, "headers": resp.Header}
}

func httpHead(url string) map[string]interface{} {
	resp, err := http.Head(url)
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	return map[string]interface{}{"headers": resp.Header, "status": resp.StatusCode}
}

func httpPost(url string, body string) map[string]interface{} {
	client := http.Client{}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	resp, err := client.Do(request)
	if err != nil {
		return map[string]interface{}{"error": err}
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	return map[string]interface{}{"body": string(respBody), "status": resp.StatusCode, "headers": resp.Header}

}

func httpPostForm(urlString string, formData map[string]interface{}) map[string]interface{} {
	values := mapToValues(formData)
	resp, err := http.PostForm(urlString, values)
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{"error": err}
	}
	return map[string]interface{}{"body": string(body), "status": resp.StatusCode, "headers": resp.Header}
}

func httpPut(url string, body string) map[string]interface{} {
	client := http.Client{}
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	resp, err := client.Do(request)
	if err != nil {
		return map[string]interface{}{"error": err}
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	return map[string]interface{}{"body": string(respBody), "status": resp.StatusCode, "headers": resp.Header}

}

func httpDelete(url string) map[string]interface{} {
	client := http.Client{}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	resp, err := client.Do(request)
	if err != nil {
		return map[string]interface{}{"error": err}
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{"error": err}
	}

	return map[string]interface{}{"body": string(respBody), "status": resp.StatusCode, "headers": resp.Header}

}

func mapToValues(m map[string]interface{}) (values url.Values) {
	values = url.Values{}
	for key, value := range m {
		values.Set(key, fmt.Sprint(value))
	}
	return
}
