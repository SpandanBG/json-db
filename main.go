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
	data, err := readJsonFile(ctx.Param("filename"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while reading file: %s - %s\n", ctx.Param("filename"), err.Error())
		return http.StatusBadRequest, nil
	}

	return bytesToJSON(data)
}

type Instruction struct {
	Query string   `json:"query"`
	Flags string   `json:"flags"`
	Files []string `json:"files"`
}

func queryFile(ctx *gin.Context) (int, interface{}) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while reading request body: %s\n", err.Error())
		return http.StatusBadRequest, nil
	}

	var instructions []Instruction
	err = json.Unmarshal(body, &instructions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while umarshalling request body: %s\nGot: %s\n", err.Error(), string(body))
		return http.StatusBadRequest, nil
	}

	var inBuff bytes.Buffer
	var outBuff bytes.Buffer
	for _, instruction := range instructions {
		inBuff.Reset()
		_, err := inBuff.Write(outBuff.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while prepping std in for cmd: %s\n", err.Error())
			return http.StatusInternalServerError, nil
		}
		outBuff.Reset()

		args := []string{fmt.Sprintf("%s", instruction.Query)}
		if len(instruction.Flags) > 0 {
			args = append(args, instruction.Flags)
		}

		for _, file := range instruction.Files {
			args = append(args, fmt.Sprintf("./data/%s.json", file))
		}

		cmd := exec.Command("jq", args...)

		cmd.Stdin = &inBuff
		cmd.Stdout = &outBuff
		cmd.Stderr = &outBuff
		err = cmd.Run()
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"error while executing query for file: %s - %s\ncmd: %s\nerr: %s\n",
				strings.Join(instruction.Files, " "), err.Error(), cmd.String(), outBuff.String(),
			)
			return http.StatusInternalServerError, nil
		}
	}

	return bytesToJSON(outBuff.Bytes())
}

func readJsonFile(filename string) ([]byte, error) {
	file, err := os.Open("./data/" + filename + ".json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while opening file: %s - %s\n", filename, err.Error())
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

func bytesToJSON(data []byte) (int, interface{}) {
	// Handle if file is array of jsons
	if data[0] == '[' {
		var jsonData []gin.H
		err := json.Unmarshal(data, &jsonData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while unmarshalling: %s\nData: %s\n", err.Error(), string(data))
			return http.StatusInternalServerError, nil
		}

		return http.StatusOK, jsonData
	}

	if data[0] == '{' {
		// Default case when file is a single json object
		var jsonData gin.H
		err := json.Unmarshal(data, &jsonData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while unmarshalling: %s\nData: %s\n", err.Error(), string(data))
			return http.StatusInternalServerError, nil
		}

		return http.StatusOK, jsonData
	}

	return http.StatusOK, string(data)
}
