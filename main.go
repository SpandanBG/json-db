package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

type jsonHandlerFn func(ctx *gin.Context) (int, interface{})

func main() {
	r := gin.Default()
	registerRoutes(r)
	r.Run()
}

func registerRoutes(r *gin.Engine) {
	r.GET("/:filename/get", jsonHandler(getFile))
	r.POST("/query", jsonHandler(queryFile))
}

func jsonHandler(handler jsonHandlerFn) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		status, data := handler(ctx)
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.JSON(int(status), data)
	}
}

func getFile(ctx *gin.Context) (int, interface{}) {
	filename := ctx.Param("filename")

	data, err := readJsonFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while reading file: %s - %s", filename, err.Error())
		return http.StatusBadRequest, nil
	}

	return bytesToJSON(filename, data)
}

type Instruction struct {
	Query string   `json:"query"`
	Files []string `json:"files"`
}

func queryFile(ctx *gin.Context) (int, interface{}) {
	filename := ctx.Param("filename")

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while reading request body: %s", err.Error())
		return http.StatusBadRequest, nil
	}

	var instruction Instruction
	err = json.Unmarshal(body, &instruction)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while umarshalling request body: %s", err.Error())
		return http.StatusBadRequest, nil
	}

	var filePaths []string
	for _, file := range instruction.Files {
		filePaths = append(filePaths, fmt.Sprintf("./data/%s.json", file))
	}

	var out bytes.Buffer
	cmd := exec.Command("jq", instruction.Query, strings.Join(filePaths, " "))

	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"error while executing query for file: %s - %s\ncmd: %s\nerr: %s\n",
			filename, err.Error(), cmd.String(), out.String(),
		)
		return http.StatusInternalServerError, nil
	}

	return bytesToJSON(filename, out.Bytes())
}

func readJsonFile(filename string) ([]byte, error) {
	file, err := os.Open("./data/" + filename + ".json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while opening file: %s - %s", filename, err.Error())
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

func bytesToJSON(filename string, data []byte) (int, interface{}) {
	// Handle if file is array of jsons
	if data[0] == '[' {
		var jsonData []gin.H
		err := json.Unmarshal(data, &jsonData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while unmarshalling file: %s - %s", filename, err.Error())
			return http.StatusInternalServerError, nil
		}

		return http.StatusOK, jsonData
	}

	// Default case when file is a single json object
	var jsonData gin.H
	err := json.Unmarshal(data, &jsonData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while unmarshalling file: %s - %s", filename, err.Error())
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, jsonData
}
