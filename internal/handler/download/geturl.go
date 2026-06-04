package download

import (
	"fmt"
	"honoka-chan/config"
	"honoka-chan/internal/middleware"
	"honoka-chan/internal/router"
	downloadschema "honoka-chan/internal/schema/download"
	"honoka-chan/internal/session"
	honokautils "honoka-chan/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func getUrl(ctx *gin.Context) {
	ss := session.Get(ctx)
	defer ss.Finalize()

	downloadReq := downloadschema.UrlReq{}
	err := honokautils.ParseRequestData(ctx, &downloadReq)
	if ss.CheckErr(err) {
		return
	}

	cdnServer := config.Conf.Settings.BackupCdnServer
	if cdnServer == "" {
		cdnServer = config.Conf.Settings.CdnServer
	}

	urlList := []string{}
	for _, v := range downloadReq.PathList {
		urlList = append(urlList, fmt.Sprintf("%s/%s/extracted/%s",
			cdnServer, downloadReq.Os, strings.ReplaceAll(v, "\\", "")))
	}

	ss.Respond(downloadschema.UrlResp{
		ResponseData: downloadschema.UrlData{
			UrlList: urlList,
		},
		ReleaseInfo: []any{},
		StatusCode:  200,
	})
}

func init() {
	router.AddHandler("main.php", "POST", "/download/getUrl", middleware.Common, getUrl)
}
