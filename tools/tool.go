package tools

import (
	"crypto/aes"
	"io"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"crypto/rand"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"os/exec"
	"bytes"
	"strings"
	"crypto/md5"
	"encoding/hex"
)

func InSliceStr(slice []string, findval string) (bool, int) {
	for i, val := range slice {
		if val == findval {
			return true, i
		}
	}
	return false, 0
}

// encrypt string to base64 crypto using AES
func Encrypt(key []byte, text string) string {
	plaintext := []byte(text)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext)
}

// decrypt from base64 to decrypted string
func Decrypt(key []byte, cryptoText string) string {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)
	return fmt.Sprintf("%s", ciphertext)
}

func KeyGen(origin string) []byte {
	for len(origin) < 32 {
		origin = origin + "1"
	}
	return []byte(origin)
}

func GetIP(botServerIP *string) {
	response, _ := http.Get("https://api.ipify.org/?format=json")
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	var data map[string]interface{}
	json.Unmarshal(body, &data)
	*botServerIP = data["ip"].(string)
}

// something like https://bittrex.com/Market/Index?MarketName=BTC-GBYTE
func GenBittrexCoinLink(marketName string) (link string) {
	return "https://bittrex.com/Market/Index?MarketName=" + marketName
}

func ExeCmd(command string) {
	fmt.Println("||| ExeCmd command is ", command)
	cmd := exec.Command("bash", "-c", command)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if !strings.Contains(fmt.Sprintln(err), "exit status 2") {
			fmt.Println(fmt.Sprintf("||| ExeCmd err = %v, : %s\n", err, stderr.String()))
			panic(err)
		}
	}
	fmt.Println("||| ExeCmd Result: " + out.String())
}

// based on https://gist.github.com/sergiotapia/8263278
func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func RemoveDuplicatesFromStrSlice(a []string) []string {
	result := []string{}
	seen := map[string]string{}
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = val
		}
	}
	return result
}