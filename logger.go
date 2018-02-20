/*******************************************************************************
 * Copyright 2017 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *
 * @microservice: support-logging-client-go library
 * @author: Ryan Comer, Dell
 * @version: 0.5.0
 *******************************************************************************/
package logger

// Logging client for the Go implementation of edgexfoundry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/support-domain-go"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type LoggingClient struct {
	OwningServiceName string
	RemoteEnabled     bool
	LogTarget         string
	StdOutLogger      *log.Logger
	FileLogger        *log.Logger
}

// Create a new logging client for the owning service
func NewClient(owningServiceName string, isRemote bool, logTarget string) LoggingClient {
	// Set up logging client
	lc := LoggingClient{
		OwningServiceName: owningServiceName,
		RemoteEnabled:     isRemote,
		LogTarget:         logTarget,
	}

	// Set up the loggers
	lc.StdOutLogger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	lc.FileLogger = &log.Logger{}
	lc.FileLogger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return lc
}

// Send the log out as a REST request
func (lc LoggingClient) log(logLevel support_domain.LogLevel, msg string, labels []string) error {

    if !lc.RemoteEnabled {
		// Save to logging file if path was set
		return lc.saveToLogFile(string(logLevel), msg)
	}

	// Send to logging service
	logEntry := lc.buildLogEntry(logLevel, msg, labels)
	return lc.sendLog(logEntry)
}

func (lc LoggingClient) saveToLogFile(prefix string, message string) error {
	if lc.LogTarget == "" {
		return nil
	}

	verifyLogDirectory(lc.LogTarget)
	file, err := os.OpenFile(lc.LogTarget, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	if err != nil {
		fmt.Println("Error opening log file: " + err.Error())
		return err
	}

	lc.FileLogger.SetOutput(file)
	lc.FileLogger.SetPrefix(prefix + ": ")
	lc.FileLogger.Println(message)

	return nil
}

func verifyLogDirectory(path string) {
	prefix, _ := filepath.Split(path)
	//If a path to the log file was specified and it does not exist, create it.
	dir := strings.TrimRight(prefix, "/")
	if len(dir) > 0 {
		if _, err := os.Stat(dir); os.IsNotExist(err){
			fmt.Println("Creating directory: " + dir)
			os.MkdirAll(dir, 0766)
		}
	}
}

// Log an INFO level message
func (lc LoggingClient) Info(msg string, labels ...string) error {
	lc.StdOutLogger.SetPrefix("INFO: ")
	lc.StdOutLogger.Println(msg)
	return lc.log(support_domain.INFO, msg, labels)
}

// Log a TRACE level message
func (lc LoggingClient) Trace(msg string, labels ...string) {

}

// Log a DEBUG level message
func (lc LoggingClient) Debug(msg string, labels ...string) error {
	lc.StdOutLogger.SetPrefix("DEBUG: ")
	lc.StdOutLogger.Println(msg)
	return lc.log(support_domain.DEBUG, msg, labels)
}

// Log a WARN level message
func (lc LoggingClient) Warn(msg string, labels ...string) error {
	lc.StdOutLogger.SetPrefix("WARN: ")
	lc.StdOutLogger.Println(msg)
	return lc.log(support_domain.WARN, msg, labels)
}

// Log an ERROR level message
func (lc LoggingClient) Error(msg string, labels ...string) error {
	lc.StdOutLogger.SetPrefix("ERROR: ")
	lc.StdOutLogger.Println(msg)
	return lc.log(support_domain.ERROR, msg, labels)
}

// Build the log entry object
func (lc LoggingClient) buildLogEntry(logLevel support_domain.LogLevel, msg string, labels []string) support_domain.LogEntry {
	res := support_domain.LogEntry{}
	res.Level = logLevel
	res.Message = msg
	res.Labels = labels
	res.OriginService = lc.OwningServiceName

	return res
}

// Send the log as an http request
func (lc LoggingClient) sendLog(logEntry support_domain.LogEntry) error {
	if lc.LogTarget == "" {
		return nil
	}

	reqBody, err := json.Marshal(logEntry)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	req, err := http.NewRequest("POST", lc.LogTarget, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	// Asynchronous call
	go lc.makeRequest(client, req)

	return nil
}

// Function to call in a goroutine
func (lc LoggingClient) makeRequest(client *http.Client, request *http.Request) {
	resp, err := client.Do(request)
	if err == nil {
		defer resp.Body.Close()
		resp.Close = true
	} else {
		fmt.Println(err.Error())
	}
}
