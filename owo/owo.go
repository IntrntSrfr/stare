package owo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

type OWOClient struct {
	token  string
	client *http.Client
}

func NewOWOClient(tkn string) *OWOClient {
	return &OWOClient{
		token:  tkn,
		client: &http.Client{},
	}
}

func (o *OWOClient) Upload(text string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="files[]"; filename="text.txt"`)
	h.Set("Content-Type", "text/plain;charset=utf-8")

	part, err := writer.CreatePart(h)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, bytes.NewReader([]byte(text)))
	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.awau.moe/upload/pomf", body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", o.token)

	res, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	resbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	jeff := OWOResult{}
	err = json.Unmarshal(resbody, &jeff)
	if err != nil {
		return "", err
	}

	if !jeff.Success {
		return "", errors.New(jeff.Description)
	}

	if len(jeff.Files) > 0 {
		return "https://chito.ge/" + jeff.Files[0].URL, nil
	}
	return "", nil
}
