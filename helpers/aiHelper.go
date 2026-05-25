package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var API_URL = os.Getenv("HUGGINGFACE_TOKEN")
func GetEmbedding(text string) ([]float32, error) {
	hfToken := os.Getenv("HUGGINGFACE_TOKEN")
	if hfToken == "" {
		return nil, fmt.Errorf("thiếu HUGGINGFACE_TOKEN trong file .env")
	}

	// HuggingFace nhận payload là 1 mảng các chuỗi (mình chỉ gửi 1 chuỗi)
	payload := map[string][]string{"inputs": {text}}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", API_URL, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+hfToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// API trả về mảng 2 chiều (vì mình truyền vào mảng chuỗi), nhưng ta chỉ cần phần tử đầu tiên
	var embeddings [][]float32
	if err := json.Unmarshal(body, &embeddings); err != nil {
		return nil, fmt.Errorf("không thể giải mã vector: %v", string(body))
	}

	if len(embeddings) > 0 {
		return embeddings[0], nil
	}
	return nil, fmt.Errorf("không nhận được vector từ AI")
}