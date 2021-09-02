// Copyright Verizon Media. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
// vespa deploy API
// Author: bratseth

package vespa

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vespa-engine/vespa/util"
)

type ApplicationID struct {
	Tenant      string
	Application string
	Instance    string
}

type ZoneID struct {
	Environment string
	Region      string
}

type Deployment struct {
	Application ApplicationID
	Zone        ZoneID
}

type DeploymentOpts struct {
	ApplicationPackage ApplicationPackage
	Target             Target
	Deployment         Deployment
	APIKey             []byte
}

type ApplicationPackage struct {
	Path string
}

func (a ApplicationID) String() string {
	return fmt.Sprintf("%s.%s.%s", a.Tenant, a.Application, a.Instance)
}

func (a ApplicationID) SerializedForm() string {
	return fmt.Sprintf("%s:%s:%s", a.Tenant, a.Application, a.Instance)
}

func (d Deployment) String() string {
	return fmt.Sprintf("deployment of %s in %s", d.Application, d.Zone)
}

func (d DeploymentOpts) String() string {
	return fmt.Sprintf("%s to %s", d.Deployment, d.Target.Type())
}

func (d *DeploymentOpts) IsCloud() bool { return d.Target.Type() == cloudTargetType }

func (d *DeploymentOpts) url(path string) (*url.URL, error) {
	service, err := d.Target.Service("deploy")
	if err != nil {
		return nil, err
	}
	return url.Parse(service.BaseURL + path)
}

func (ap *ApplicationPackage) HasCertificate() bool {
	if ap.IsZip() {
		return true // TODO: Consider looking inside zip to verify
	}
	return util.PathExists(filepath.Join(ap.Path, "security", "clients.pem"))
}

func (ap *ApplicationPackage) IsZip() bool { return isZip(ap.Path) }

func (ap *ApplicationPackage) zipReader() (io.ReadCloser, error) {
	zipFile := ap.Path
	if !ap.IsZip() {
		tempZip, error := ioutil.TempFile("", "application.zip")
		if error != nil {
			return nil, fmt.Errorf("Could not create a temporary zip file for the application package: %w", error)
		}
		if err := zipDir(ap.Path, tempZip.Name()); err != nil {
			return nil, err
		}
		defer os.Remove(tempZip.Name())
		zipFile = tempZip.Name()
	}
	r, err := os.Open(zipFile)
	if err != nil {
		return nil, fmt.Errorf("Could not open application package at %s: %w", ap.Path, err)
	}
	return r, nil
}

// Find an application package zip or directory below an application path
func ApplicationPackageFrom(application string) (ApplicationPackage, error) {
	if isZip(application) {
		return ApplicationPackage{Path: application}, nil
	}
	if util.PathExists(filepath.Join(application, "pom.xml")) {
		zip := filepath.Join(application, "target", "application.zip")
		if !util.PathExists(zip) {
			return ApplicationPackage{}, errors.New("pom.xml exists but no target/application.zip. Run mvn package first")
		} else {
			return ApplicationPackage{Path: zip}, nil
		}
	}
	if util.PathExists(filepath.Join(application, "src", "main", "application")) {
		return ApplicationPackage{Path: filepath.Join(application, "src", "main", "application")}, nil
	}
	if util.PathExists(filepath.Join(application, "services.xml")) {
		return ApplicationPackage{Path: application}, nil
	}
	return ApplicationPackage{}, errors.New("Could not find an application package source in '" + application + "'")
}

func ApplicationFromString(s string) (ApplicationID, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return ApplicationID{}, fmt.Errorf("invalid application: %q", s)
	}
	return ApplicationID{Tenant: parts[0], Application: parts[1], Instance: parts[2]}, nil
}

func ZoneFromString(s string) (ZoneID, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return ZoneID{}, fmt.Errorf("invalid zone: %q", s)
	}
	return ZoneID{Environment: parts[0], Region: parts[1]}, nil
}

// Prepare deployment and return the session ID
func Prepare(deployment DeploymentOpts) (int64, error) {
	if deployment.IsCloud() {
		return 0, fmt.Errorf("%s: prepare is not supported", deployment)
	}
	sessionURL, err := deployment.url("/application/v2/tenant/default/session")
	if err != nil {
		return 0, err
	}
	sessionID, err := uploadApplicationPackage(sessionURL, deployment)
	if err != nil {
		return 0, err
	}
	prepareURL, err := deployment.url(fmt.Sprintf("/application/v2/tenant/default/session/%d/prepared", sessionID))
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest("PUT", prepareURL.String(), nil)
	if err != nil {
		return 0, err
	}
	serviceDescription := "Deploy service"
	response, err := util.HttpDo(req, time.Second*30, serviceDescription)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if err := checkResponse(req, response, serviceDescription); err != nil {
		return 0, err
	}
	return sessionID, nil
}

// Activate deployment with sessionID from a past prepare
func Activate(sessionID int64, deployment DeploymentOpts) error {
	if deployment.IsCloud() {
		return fmt.Errorf("%s: activate is not supported", deployment)
	}
	u, err := deployment.url(fmt.Sprintf("/application/v2/tenant/default/session/%d/active", sessionID))
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", u.String(), nil)
	if err != nil {
		return err
	}
	serviceDescription := "Deploy service"
	response, err := util.HttpDo(req, time.Second*30, serviceDescription)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return checkResponse(req, response, serviceDescription)
}

func Deploy(opts DeploymentOpts) error {
	path := "/application/v2/tenant/default/prepareandactivate"
	if opts.IsCloud() {
		if !opts.ApplicationPackage.HasCertificate() {
			return fmt.Errorf("%s: missing certificate in package", opts)
		}
		if opts.APIKey == nil {
			return fmt.Errorf("%s: missing api key", opts.String())
		}
		if opts.Deployment.Zone.Environment == "" || opts.Deployment.Zone.Region == "" {
			return fmt.Errorf("%s: missing zone", opts)
		}
		path = fmt.Sprintf("/application/v4/tenant/%s/application/%s/instance/%s/deploy/%s-%s",
			opts.Deployment.Application.Tenant,
			opts.Deployment.Application.Application,
			opts.Deployment.Application.Instance,
			opts.Deployment.Zone.Environment,
			opts.Deployment.Zone.Region)
	}
	u, err := opts.url(path)
	if err != nil {
		return err
	}
	_, err = uploadApplicationPackage(u, opts)
	return err
}

func uploadApplicationPackage(url *url.URL, opts DeploymentOpts) (int64, error) {
	zipReader, err := opts.ApplicationPackage.zipReader()
	if err != nil {
		return 0, err
	}
	header := http.Header{}
	header.Add("Content-Type", "application/zip")
	request := &http.Request{
		URL:    url,
		Method: "POST",
		Header: header,
		Body:   io.NopCloser(zipReader),
	}
	if opts.APIKey != nil {
		signer := NewRequestSigner(opts.Deployment.Application.SerializedForm(), opts.APIKey)
		if err := signer.SignRequest(request); err != nil {
			return 0, err
		}
	}
	serviceDescription := "Deploy service"
	response, err := util.HttpDo(request, time.Minute*10, serviceDescription)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	var sessionResponse struct {
		SessionID string `json:"session-id"`
	}
	sessionResponse.SessionID = "0" // Set a default session ID for responses that don't contain int (e.g. cloud deployment)
	if err := checkResponse(request, response, serviceDescription); err != nil {
		return 0, err
	}
	jsonDec := json.NewDecoder(response.Body)
	jsonDec.Decode(&sessionResponse) // Ignore error in case this is a non-JSON response
	return strconv.ParseInt(sessionResponse.SessionID, 10, 64)
}

func checkResponse(req *http.Request, response *http.Response, serviceDescription string) error {
	if response.StatusCode/100 == 4 {
		return fmt.Errorf("Invalid application package (%s)\n\n%s", response.Status, extractError(response.Body))
	} else if response.StatusCode != 200 {
		return fmt.Errorf("Error from %s at %s (%s):\n%s", strings.ToLower(serviceDescription), req.URL.Host, response.Status, util.ReaderToJSON(response.Body))
	}
	return nil
}

func isZip(filename string) bool { return filepath.Ext(filename) == ".zip" }

func zipDir(dir string, destination string) error {
	if filepath.IsAbs(dir) {
		message := "Path must be relative, but '" + dir + "'"
		return errors.New(message)
	}
	if !util.PathExists(dir) {
		message := "'" + dir + "' should be an application package zip or dir, but does not exist"
		return errors.New(message)
	}
	if !util.IsDirectory(dir) {
		message := "'" + dir + "' should be an application package dir, but is a (non-zip) file"
		return errors.New(message)
	}

	file, err := os.Create(destination)
	if err != nil {
		message := "Could not create a temporary zip file for the application package: " + err.Error()
		return errors.New(message)
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		zippath := strings.TrimPrefix(path, dir)
		zipfile, err := w.Create(zippath)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipfile, file)
		if err != nil {
			return err
		}
		return nil
	}
	return filepath.Walk(dir, walker)
}

// Returns the error message in the given JSON, or the entire content if it could not be extracted
func extractError(reader io.Reader) string {
	responseData := util.ReaderToBytes(reader)
	var response map[string]interface{}
	json.Unmarshal(responseData, &response)
	if response["error-code"] == "INVALID_APPLICATION_PACKAGE" {
		return strings.ReplaceAll(response["message"].(string), ": ", ":\n")
	} else {
		var prettyJSON bytes.Buffer
		parseError := json.Indent(&prettyJSON, responseData, "", "    ")
		if parseError != nil { // Not JSON: Print plainly
			return string(responseData)
		}
		return prettyJSON.String()
	}
}
