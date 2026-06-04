package download

import (
	"fmt"
	"honoka-chan/config"
	"honoka-chan/internal/middleware"
	"honoka-chan/internal/router"
	downloadschema "honoka-chan/internal/schema/download"
	"honoka-chan/internal/session"
	honokautils "honoka-chan/internal/utils"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const overrideServerConfigFileName = "99_0_115.zip"

func update(ctx *gin.Context) {
	ss := session.Get(ctx)
	defer ss.Finalize()

	downloadReq := downloadschema.UpdateReq{}
	err := honokautils.ParseRequestData(ctx, &downloadReq)
	if ss.CheckErr(err) {
		return
	}

	pkgList := []downloadschema.UpdateData{}
	if downloadReq.ExternalVersion != config.PackageVersion {
		pkgType := 99
		var pkgInfo []PkgInfo
		err := ss.MainEng.Table("download_m").Where("pkg_type = ? AND pkg_os = ?", pkgType, downloadReq.TargetOs).
			OrderBy("pkg_id ASC, pkg_order ASC").Find(&pkgInfo)
		if ss.CheckErr(err) {
			return
		}

		pkgMap := map[string]int{}
		for _, pkg := range pkgInfo {
			fileName := fmt.Sprintf("%d_%d_%d.zip", pkg.PkgType, pkg.PkgID, pkg.Order)
			url := fmt.Sprintf("%s/%s/archives/%s",
				config.Conf.Settings.CdnServer, downloadReq.TargetOs, fileName)

			pkgList = append(pkgList, downloadschema.UpdateData{
				Size:    pkg.Size,
				URL:     url,
				Version: config.PackageVersion,
			})
			pkgMap[fileName] = len(pkgList) - 1
		}

		overrideSource := getOverrideSource(downloadReq.TargetOs)
		serverConfigURL := fmt.Sprintf("%s/%s/archives/%s",
			config.Conf.Settings.CdnServer, downloadReq.TargetOs, overrideServerConfigFileName)
		serverConfigURL, serverConfigSize, serverConfigAvailable := resolveServerConfigFile(serverConfigURL, overrideSource)

		if serverConfigAvailable {
			if index, ok := pkgMap[overrideServerConfigFileName]; ok {
				pkgList[index].URL = serverConfigURL
				pkgList[index].Size = serverConfigSize
			} else {
				pkgList = append(pkgList, downloadschema.UpdateData{
					Size:    serverConfigSize,
					URL:     serverConfigURL,
					Version: config.PackageVersion,
				})
			}
		}

		applyOverrideFileSize(downloadReq.TargetOs, pkgList)
	}

	ss.Respond(downloadschema.UpdateResp{
		ResponseData: pkgList,
		ReleaseInfo:  []any{},
		StatusCode:   200,
	})
}

func getOverrideSource(targetOs string) config.OverrideFileSource {
	if strings.EqualFold(targetOs, "Android") {
		return config.Conf.Settings.OverrideServerConfig.Android
	}
	if strings.EqualFold(targetOs, "iOS") {
		return config.Conf.Settings.OverrideServerConfig.IOS
	}
	return config.OverrideFileSource{}
}

func resolveServerConfigFile(defaultURL string, overrideSource config.OverrideFileSource) (string, int, bool) {
	if config.Conf.Settings.OverrideServerConfig.Enable && overrideSource.URL != "" {
		size, ok := getRemoteFileSize(overrideSource.URL)
		if !ok {
			return overrideSource.URL, 0, false
		}
		if overrideSource.Size > 0 {
			size = overrideSource.Size
		}
		return overrideSource.URL, size, true
	}

	size, ok := getRemoteFileSize(defaultURL)
	return defaultURL, size, ok
}

func getRemoteFileSize(url string) (int, bool) {
	client := &http.Client{Timeout: 5 * time.Second}

	headReq, err := http.NewRequest(http.MethodHead, url, nil)
	if err == nil {
		if resp, err := client.Do(headReq); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
				if resp.ContentLength > 0 {
					return int(resp.ContentLength), true
				}
			}
		}
	}

	getResp, err := client.Get(url)
	if err != nil {
		return 0, false
	}
	defer getResp.Body.Close()
	if getResp.StatusCode < http.StatusOK || getResp.StatusCode >= http.StatusMultipleChoices {
		return 0, false
	}

	dataLen, err := io.Copy(io.Discard, getResp.Body)
	if err != nil {
		return 0, false
	}

	return int(dataLen), dataLen > 0
}

func applyOverrideFileSize(targetOs string, pkgList []downloadschema.UpdateData) {
	urlSizeCache := map[string]int{}

	for i := range pkgList {
		fileName := getFileNameFromURL(pkgList[i].URL)
		if fileName == "" {
			continue
		}

		for _, override := range config.Conf.Settings.OverrideFileSize {
			overrideOs := strings.TrimSpace(override.TargetOs)
			if overrideOs != "" && !strings.EqualFold(overrideOs, strings.TrimSpace(targetOs)) {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(override.FileName), fileName) {
				size, ok := urlSizeCache[pkgList[i].URL]
				if !ok {
					size, _ = getRemoteFileSize(pkgList[i].URL)
					urlSizeCache[pkgList[i].URL] = size
				}

				if size > 0 {
					pkgList[i].Size = size
				}
				break
			}
		}
	}
}

func getFileNameFromURL(rawURL string) string {
	urlNoQuery := strings.Split(rawURL, "?")[0]
	parts := strings.Split(urlNoQuery, "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func init() {
	router.AddHandler("main.php", "POST", "/download/update", middleware.Common, update)
}
