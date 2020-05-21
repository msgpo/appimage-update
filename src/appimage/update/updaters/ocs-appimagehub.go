package updaters

import (
	"appimage-update/src/appimage"
	"bufio"
	"bytes"
	"fmt"
	"github.com/beevik/etree"
	"github.com/danwakefield/fnmatch"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"strings"
)

type OCSAppImageHub struct {
	direct Direct

	apiV1Url  string
	productId string
	fileName  string
}

func NewOCSAppImageHub(updateInfoString *string, target *appimage.AppImage) (*OCSAppImageHub, error) {
	parts := strings.Split(*updateInfoString, "|")

	if len(parts) != 4 {
		return nil, fmt.Errorf("Invalid OCSAppImageHub update instance. Expected: ocs-v1-appimagehub-direct|<api url>|<product id>|<file name>")
	}

	instance := OCSAppImageHub{
		direct: Direct{
			seed: *target,
		},

		apiV1Url:  parts[1],
		productId: parts[2],
		fileName:  parts[3],
	}

	return &instance, nil
}

func (O *OCSAppImageHub) Method() string {
	return "ocs-v1-appimagehub-direct"
}

func (O *OCSAppImageHub) Lookup() (updateAvailable bool, err error) {
	url := fmt.Sprint("https://", O.apiV1Url, "/content/data/", O.productId)
	data, err := getContentData(url)
	if err != nil {
		return false, err
	}

	doc := etree.NewDocument()
	err = doc.ReadFromBytes(data)
	if err != nil {
		return false, err
	}

	O.direct.url = O.resolveDownloadUrl(doc)
	return O.direct.Lookup()
}

func (O *OCSAppImageHub) resolveDownloadUrl(doc *etree.Document) string {
	downloadIdx := 1
	for true {
		downloadNameTag := fmt.Sprintf("//downloadname%d", downloadIdx)
		downloadNameItem := doc.FindElement(downloadNameTag)
		if downloadNameItem == nil {
			break
		}
		downloadName := downloadNameItem.Text()

		if fnmatch.Match(O.fileName, downloadName, fnmatch.FNM_IGNORECASE) {
			downloadLinkTag := fmt.Sprintf("//downloadlink%d", downloadIdx)
			return doc.FindElement(downloadLinkTag).Text()
		}

		downloadIdx++
	}

	return ""
}

func (O *OCSAppImageHub) Download() (output string, err error) {
	return O.direct.Download()
}

func getContentData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading content data: "+url,
	)

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	_, err = io.Copy(io.MultiWriter(bufWriter, bar), resp.Body)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
