package api

import (
	"honoka-chan/internal/constant"
	"honoka-chan/internal/handler/api/achievement"
	"honoka-chan/internal/handler/api/album"
	"honoka-chan/internal/handler/api/award"
	"honoka-chan/internal/handler/api/background"
	"honoka-chan/internal/handler/api/banner"
	"honoka-chan/internal/handler/api/challenge"
	"honoka-chan/internal/handler/api/costume"
	"honoka-chan/internal/handler/api/eventscenario"
	"honoka-chan/internal/handler/api/exchange"
	"honoka-chan/internal/handler/api/item"
	"honoka-chan/internal/handler/api/live"
	"honoka-chan/internal/handler/api/liveicon"
	"honoka-chan/internal/handler/api/livese"
	"honoka-chan/internal/handler/api/login"
	"honoka-chan/internal/handler/api/marathon"
	"honoka-chan/internal/handler/api/multiunit"
	"honoka-chan/internal/handler/api/museum"
	"honoka-chan/internal/handler/api/navigation"
	"honoka-chan/internal/handler/api/notice"
	"honoka-chan/internal/handler/api/payment"
	"honoka-chan/internal/handler/api/profile"
	"honoka-chan/internal/handler/api/reward"
	"honoka-chan/internal/handler/api/scenario"
	"honoka-chan/internal/handler/api/secretbox"
	"honoka-chan/internal/handler/api/stamp"
	"honoka-chan/internal/handler/api/subscenario"
	"honoka-chan/internal/handler/api/unit"
	"honoka-chan/internal/handler/api/user"
	"honoka-chan/internal/middleware"
	"honoka-chan/internal/router"
	apischema "honoka-chan/internal/schema/api"
	commonschema "honoka-chan/internal/schema/common"
	"honoka-chan/internal/session"
	honokautils "honoka-chan/internal/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func api(ctx *gin.Context) {
	ss := session.Get(ctx)
	defer ss.FinalizeOrRollback()

	apiReq := []apischema.ApiReq{}
	err := honokautils.ParseRequestData(ctx, &apiReq)
	if ss.CheckErr(err) {
		return
	}

	results := []any{}
	for _, v := range apiReq {
		var result any
		var itemErr error

		switch v.Module {
		case "album":
			result, itemErr = album.AlbumApi(ctx, v.Action)
		case "achievement":
			result, itemErr = achievement.AchievementApi(ctx, v.Action)
		case "award":
			result, itemErr = award.AwardApi(ctx, v.Action)
		case "background":
			result, itemErr = background.BackgroundApi(ctx, v.Action)
		case "banner":
			result, itemErr = banner.BannerApi(v.Action)
		case "challenge":
			result, itemErr = challenge.ChallengeApi(v.Action)
		case "costume":
			result, itemErr = costume.CostumeApi(v.Action)
		case "eventscenario":
			result, itemErr = eventscenario.EventScenarioApi(ctx, v.Action)
		case "exchange":
			result, itemErr = exchange.ExchangeApi(ctx, v.Action)
		case "item":
			result, itemErr = item.ItemApi(v.Action)
		case "live":
			result, itemErr = live.LiveApi(ctx, v.Action)
		case "liveicon":
			result, itemErr = liveicon.LiveIconApi(v.Action)
		case "livese":
			result, itemErr = livese.LiveSeApi(v.Action)
		case "login":
			result, itemErr = login.LoginApi(ctx, v.Action)
		case "marathon":
			result, itemErr = marathon.MarathonApi(v.Action)
		case "multiunit":
			result, itemErr = multiunit.MultiUnitApi(ctx, v.Action)
		case "museum":
			result, itemErr = museum.MuseumApi(ctx, v.Action)
		case "navigation":
			result, itemErr = navigation.NavigationApi(v.Action)
		case "notice":
			result, itemErr = notice.NoticeApi(v.Action)
		case "payment":
			result, itemErr = payment.PaymentApi(ctx, v.Action)
		case "profile":
			result, itemErr = profile.ProfileApi(ctx, v.Action, v.UserID)
		case "reward":
			result, itemErr = reward.RewardApi(v.Action)
		case "scenario":
			result, itemErr = scenario.ScenarioApi(ctx, v.Action)
		case "secretbox":
			result, itemErr = secretbox.SecretBoxApi(v.Action)
		case "stamp":
			result, itemErr = stamp.StampApi(v.Action)
		case "subscenario":
			result, itemErr = subscenario.SubscenarioApi(ctx, v.Action)
		case "unit":
			result, itemErr = unit.UnitApi(ctx, v.Action)
		case "user":
			result, itemErr = user.UserApi(ctx, v.Action)
		default:
			itemErr = honokautils.NewUnimplementedModuleError(v.Module)
		}

		// Keep the mainline module implementations and signatures intact, but
		// preserve the Android branch's proven per-item fallback for an unknown
		// module/action. One unsupported item must not abort the entire batch.
		if honokautils.IsUnimplementedError(itemErr) {
			results = append(results, commonschema.ApiErrorResp{
				Result: commonschema.ErrorData{
					ErrorCode: constant.ErrorCode(1),
				},
				Status:     constant.ErrorCodeAcceptableError,
				CommandNum: false,
				TimeStamp:  time.Now().Unix(),
			})
			continue
		}
		if ss.CheckErr(itemErr) {
			return
		}
		results = append(results, result)
	}

	apiResp := apischema.ApiResp{
		ResponseData: results,
		ReleaseInfo:  []any{},
		StatusCode:   http.StatusOK,
	}

	ss.Respond(apiResp)
}

func init() {
	router.AddHandler("main.php", "POST", "/api", middleware.Common, api)
}
