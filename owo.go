package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

type OWOClient struct {
	Token string
}

func NewOWOClient(token string) *OWOClient {
	return &OWOClient{
		Token: token,
	}
}

func (o *OWOClient) Upload(text string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="files[]" filename="text.txt"`)
	h.Set("Content-Type", "text/plain")

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
	req.Header.Set("Authorization", o.Token)

	cl := http.Client{}
	res, err := cl.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	resbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	jeff := OWOResult{}
	json.Unmarshal(resbody, &jeff)

	if len(jeff.Files) > 0 {
		return "https://owo.whats-th.is/" + jeff.Files[0].URL, nil
	}
	return "", nil
}
