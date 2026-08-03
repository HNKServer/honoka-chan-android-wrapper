package startup

import (
	"encoding/json"
	"errors"
	"fmt"
	"honoka-chan/internal/constant"
	ghomemodel "honoka-chan/internal/model/ghome"
	loginmodel "honoka-chan/internal/model/login"
	unitmodel "honoka-chan/internal/model/unit"
	usermodel "honoka-chan/internal/model/user"
	"honoka-chan/pkg/db"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"xorm.io/xorm"
)

var (
	userEng *xorm.Session
)

func CreateTables() {
	// db.UserEng.ShowSQL(true)
	db.UserEng.Sync2(new(ghomemodel.DeviceKey))
	db.UserEng.Sync2(new(loginmodel.AuthKey))
	db.UserEng.Sync2(new(usermodel.UserAccessory))
	db.UserEng.Sync2(new(usermodel.UserAccessoryWear))
	db.UserEng.Sync2(new(usermodel.UserDeck))
	db.UserEng.Sync2(new(usermodel.UserDeckUnit))
	db.UserEng.Sync2(new(usermodel.UserKey))
	db.UserEng.Sync2(new(usermodel.UserLiveGoal))
	db.UserEng.Sync2(new(usermodel.UserLiveStatus))
	db.UserEng.Sync2(new(usermodel.UserLiveInProgress))
	db.UserEng.Sync2(new(usermodel.UserLiveRandom))
	db.UserEng.Sync2(new(usermodel.UserLiveRecord))
	db.UserEng.Sync2(new(usermodel.UserFriend))
	db.UserEng.Sync2(new(usermodel.UserGreet))
	db.UserEng.Sync2(new(usermodel.UserPref))
	db.UserEng.Sync2(new(usermodel.Users))
	db.UserEng.Sync2(new(usermodel.UserUnit))
	db.UserEng.Sync2(new(usermodel.UserUnitSkillEquip))

	if err := RepairLegacyUserIDColumns(); err != nil {
		log.Fatalln("迁移旧版 user_id 字段失败:", err.Error())
	}
	if err := MigrateLegacyTermuxTables(); err != nil {
		log.Fatalln("迁移旧版 Termux 用户数据表失败:", err.Error())
	}

	MigrateUserPref()
	MigrateUserAccessories()
	MigrateUserLiveData()
	ForceAllUsersRelogin()
	MigrateLegacyAuthorizeTokens()
}

type legacyWrapperUserPrefs struct {
	Name           *string `json:"name"`
	Level          *int    `json:"level"`
	ExpNumerator   *int    `json:"exp_numerator"`
	ExpDenominator *int    `json:"exp_denominator"`
	GameCoin       *int    `json:"game_coin"`
	SnsCoin        *int    `json:"sns_coin"`
	EnergyMax      *int    `json:"energy_max"`
	OverMaxEnergy  *int    `json:"over_max_energy"`
	InviteCode     *string `json:"invite_code"`
}

type legacyWrapperConfig struct {
	UserPrefs legacyWrapperUserPrefs `json:"user_prefs"`
}

type legacyUserPreference struct {
	ID               int    `xorm:"id"`
	UserID           int    `xorm:"user_id"`
	AwardID          int    `xorm:"award_id"`
	BackgroundID     int    `xorm:"background_id"`
	UnitOwningUserID int    `xorm:"unit_owning_user_id"`
	UserName         string `xorm:"user_name"`
	UserLevel        int    `xorm:"user_level"`
	UserDesc         string `xorm:"user_desc"`
	UpdateTime       int64  `xorm:"update_time"`
}

type legacyUserDeck struct {
	ID         int    `xorm:"id"`
	DeckID     int    `xorm:"deck_id"`
	MainFlag   int    `xorm:"main_flag"`
	DeckName   string `xorm:"deck_name"`
	UserID     int    `xorm:"user_id"`
	InsertDate int64  `xorm:"insert_date"`
}

type legacyDeckUnit struct {
	ID               int   `xorm:"id"`
	UserDeckID       int   `xorm:"user_deck_id"`
	UnitOwningUserID int   `xorm:"unit_owning_user_id"`
	UnitID           int   `xorm:"unit_id"`
	Position         int   `xorm:"position"`
	Level            int   `xorm:"level"`
	LevelLimitID     int   `xorm:"level_limit_id"`
	DisplayRank      int   `xorm:"display_rank"`
	Love             int   `xorm:"love"`
	UnitSkillLevel   int   `xorm:"unit_skill_level"`
	IsRankMax        int   `xorm:"is_rank_max"`
	IsLoveMax        int   `xorm:"is_love_max"`
	IsLevelMax       int   `xorm:"is_level_max"`
	IsSigned         int   `xorm:"is_signed"`
	BeforeLove       int   `xorm:"before_love"`
	MaxLove          int   `xorm:"max_love"`
	InsertDate       int64 `xorm:"insert_date"`
}

type legacyUserUnit struct {
	UnitOwningUserID            int    `xorm:"unit_owning_user_id"`
	UserID                      int    `xorm:"user_id"`
	UnitID                      int    `xorm:"unit_id"`
	Exp                         int    `xorm:"exp"`
	NextExp                     int    `xorm:"next_exp"`
	Level                       int    `xorm:"level"`
	MaxLevel                    int    `xorm:"max_level"`
	LevelLimitID                int    `xorm:"level_limit_id"`
	Rank                        int    `xorm:"rank"`
	MaxRank                     int    `xorm:"max_rank"`
	Love                        int    `xorm:"love"`
	MaxLove                     int    `xorm:"max_love"`
	UnitSkillExp                int    `xorm:"unit_skill_exp"`
	UnitSkillLevel              int    `xorm:"unit_skill_level"`
	MaxHp                       int    `xorm:"max_hp"`
	UnitRemovableSkillCapacity  int    `xorm:"unit_removable_skill_capacity"`
	FavoriteFlag                int    `xorm:"favorite_flag"`
	DisplayRank                 int    `xorm:"display_rank"`
	IsRankMax                   int    `xorm:"is_rank_max"`
	IsLoveMax                   int    `xorm:"is_love_max"`
	IsLevelMax                  int    `xorm:"is_level_max"`
	IsSigned                    int    `xorm:"is_signed"`
	IsSkillLevelMax             int    `xorm:"is_skill_level_max"`
	IsRemovableSkillCapacityMax int    `xorm:"is_removable_skill_capacity_max"`
	InsertDate                  string `xorm:"insert_date"`
}

type legacyAccessoryWear struct {
	AccessoryOwningUserID int `xorm:"accessory_owning_user_id"`
	UnitOwningUserID      int `xorm:"unit_owning_user_id"`
	UserID                int `xorm:"user_id"`
}

type legacySkillEquip struct {
	UnitOwningUserID     int `xorm:"unit_owning_user_id"`
	UnitRemovableSkillID int `xorm:"unit_removable_skill_id"`
	UserID               int `xorm:"user_id"`
}

type migratedDeckRef struct {
	TargetDeckID int
	UserID       int
}

// MigrateLegacyTermuxTables imports the tables used by the former Termux core
// into the renamed tables used by the current mainline core. The two versions
// share the same users and user_key tables, so login can appear to succeed even
// when user_preference_m was never copied to user_pref. The next userInfo request
// then fails with "用户不存在" and the client opens its maintenance page.
//
// The migration is idempotent: it only creates target rows that do not already
// exist and leaves all mainline-native rows untouched.
func MigrateLegacyTermuxTables() error {
	session := db.UserEng.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return err
	}

	legacyPrefs := loadLegacyWrapperUserPrefs()

	if err := migrateLegacyUserPreferences(session, legacyPrefs); err != nil {
		session.Rollback()
		return err
	}
	if err := migrateLegacyUserUnits(session); err != nil {
		session.Rollback()
		return err
	}
	if err := migrateLegacyDecks(session); err != nil {
		session.Rollback()
		return err
	}
	if err := migrateLegacyAccessoryWear(session); err != nil {
		session.Rollback()
		return err
	}
	if err := migrateLegacySkillEquip(session); err != nil {
		session.Rollback()
		return err
	}
	if err := ensureProfilesForExistingUsers(session, legacyPrefs); err != nil {
		session.Rollback()
		return err
	}

	if err := session.Commit(); err != nil {
		session.Rollback()
		return err
	}
	return nil
}

func loadLegacyWrapperUserPrefs() legacyWrapperUserPrefs {
	data, err := os.ReadFile("./config.json")
	if err != nil {
		return legacyWrapperUserPrefs{}
	}
	var legacy legacyWrapperConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return legacyWrapperUserPrefs{}
	}
	return legacy.UserPrefs
}

func applyLegacyWrapperProfile(pref *usermodel.UserPref, legacy legacyWrapperUserPrefs) {
	if legacy.Name != nil && strings.TrimSpace(pref.UserName) == "" {
		pref.UserName = strings.TrimSpace(*legacy.Name)
	}
	if legacy.Level != nil && *legacy.Level > 0 {
		pref.UserLevel = *legacy.Level
	}
	if legacy.ExpNumerator != nil && *legacy.ExpNumerator >= 0 {
		pref.UserExp = *legacy.ExpNumerator
	}
	if legacy.ExpDenominator != nil && *legacy.ExpDenominator > 0 {
		pref.NextExp = *legacy.ExpDenominator
	}
	if legacy.GameCoin != nil && *legacy.GameCoin >= 0 {
		pref.GameCoin = *legacy.GameCoin
	}
	if legacy.SnsCoin != nil && *legacy.SnsCoin >= 0 {
		pref.SnsCoin = *legacy.SnsCoin
	}
	if legacy.EnergyMax != nil && *legacy.EnergyMax > 0 {
		pref.EnergyMax = *legacy.EnergyMax
	}
	if legacy.OverMaxEnergy != nil && *legacy.OverMaxEnergy >= 0 {
		pref.OverMaxEnergy = *legacy.OverMaxEnergy
	}
	if legacy.InviteCode != nil && strings.TrimSpace(*legacy.InviteCode) != "" {
		pref.InviteCode = strings.TrimSpace(*legacy.InviteCode)
	}
}

func migrateLegacyUserPreferences(session *xorm.Session, wrapperPrefs legacyWrapperUserPrefs) error {
	hasLegacy, err := db.UserEng.IsTableExist("user_preference_m")
	if err != nil || !hasLegacy {
		return err
	}

	rows := []legacyUserPreference{}
	if err := session.Table("user_preference_m").Find(&rows); err != nil {
		return fmt.Errorf("读取 user_preference_m 失败: %w", err)
	}

	migrated := 0
	for _, row := range rows {
		if row.UserID <= 0 {
			continue
		}
		exists, err := session.Table(new(usermodel.UserPref)).
			Where("user_id = ?", row.UserID).Exist()
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		pref := usermodel.UserPref{
			UserID:           row.UserID,
			AwardID:          row.AwardID,
			BackgroundID:     row.BackgroundID,
			UnitOwningUserID: row.UnitOwningUserID,
			UserName:         row.UserName,
			UserLevel:        row.UserLevel,
			UserDesc:         row.UserDesc,
			InviteCode:       strconv.Itoa(row.UserID),
			UserExp:          usermodel.DefaultUserExp,
			NextExp:          usermodel.DefaultUserNextExp,
			GameCoin:         usermodel.DefaultUserGameCoin,
			SnsCoin:          usermodel.DefaultUserSnsCoin,
			EnergyMax:        usermodel.DefaultUserEnergyMax,
			OverMaxEnergy:    usermodel.DefaultUserOverMaxEnergy,
			BirthMonth:       usermodel.DefaultBirthMonth,
			BirthDay:         usermodel.DefaultBirthDay,
			ProfileVersion:   usermodel.CurrentUserPrefProfileVersion,
			UpdateTime:       row.UpdateTime,
		}
		if pref.AwardID <= 0 {
			pref.AwardID = 1
		}
		if pref.BackgroundID <= 0 {
			pref.BackgroundID = 1
		}
		if strings.TrimSpace(pref.UserName) == "" {
			pref.UserName = usermodel.DefaultAutoUserName
		}
		if strings.TrimSpace(pref.UserDesc) == "" {
			pref.UserDesc = usermodel.DefaultAutoUserDesc
		}
		if pref.UpdateTime <= 0 {
			pref.UpdateTime = time.Now().Unix()
		}
		applyLegacyWrapperProfile(&pref, wrapperPrefs)
		pref.ApplyProfileDefaults()

		if _, err := session.Insert(&pref); err != nil {
			return fmt.Errorf("迁移用户 %d 的 user_preference_m 失败: %w", row.UserID, err)
		}
		migrated++
	}
	if migrated > 0 {
		log.Printf("已从 user_preference_m 迁移 %d 个用户配置", migrated)
	}
	return nil
}

func migrateLegacyUserUnits(session *xorm.Session) error {
	hasLegacy, err := db.UserEng.IsTableExist("user_unit_m")
	if err != nil || !hasLegacy {
		return err
	}
	rows := []legacyUserUnit{}
	if err := session.Table("user_unit_m").Find(&rows); err != nil {
		return fmt.Errorf("读取 user_unit_m 失败: %w", err)
	}
	migrated := 0
	for _, row := range rows {
		if row.UserID <= 0 || row.UnitOwningUserID <= 0 {
			continue
		}
		exists, err := session.Table(new(usermodel.UserUnit)).
			Where("user_id = ? AND unit_owning_user_id = ?", row.UserID, row.UnitOwningUserID).
			Exist()
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		target := usermodel.UserUnit{
			UnitOwningUserID:            row.UnitOwningUserID,
			UserID:                      row.UserID,
			UnitID:                      row.UnitID,
			Exp:                         row.Exp,
			NextExp:                     row.NextExp,
			Level:                       row.Level,
			MaxLevel:                    row.MaxLevel,
			LevelLimitID:                row.LevelLimitID,
			Rank:                        row.Rank,
			MaxRank:                     row.MaxRank,
			Love:                        row.Love,
			MaxLove:                     row.MaxLove,
			UnitSkillExp:                row.UnitSkillExp,
			UnitSkillLevel:              row.UnitSkillLevel,
			MaxHp:                       row.MaxHp,
			UnitRemovableSkillCapacity:  row.UnitRemovableSkillCapacity,
			FavoriteFlag:                row.FavoriteFlag,
			DisplayRank:                 row.DisplayRank,
			IsRankMax:                   row.IsRankMax,
			IsLoveMax:                   row.IsLoveMax,
			IsLevelMax:                  row.IsLevelMax,
			IsSigned:                    row.IsSigned,
			IsSkillLevelMax:             row.IsSkillLevelMax,
			IsRemovableSkillCapacityMax: row.IsRemovableSkillCapacityMax,
			InsertDate:                  row.InsertDate,
		}
		if _, err := session.Insert(&target); err != nil {
			return fmt.Errorf("迁移 user_unit_m 失败: %w", err)
		}
		migrated++
	}
	if migrated > 0 {
		log.Printf("已从 user_unit_m 迁移 %d 条社员记录", migrated)
	}
	return nil
}

func migrateLegacyDecks(session *xorm.Session) error {
	hasDecks, err := db.UserEng.IsTableExist("user_deck_m")
	if err != nil || !hasDecks {
		return err
	}
	legacyDecks := []legacyUserDeck{}
	if err := session.Table("user_deck_m").Find(&legacyDecks); err != nil {
		return fmt.Errorf("读取 user_deck_m 失败: %w", err)
	}

	deckMap := make(map[int]migratedDeckRef, len(legacyDecks))
	migratedDecks := 0
	for _, row := range legacyDecks {
		if row.UserID <= 0 {
			continue
		}
		target := usermodel.UserDeck{}
		exists, err := session.Table(new(usermodel.UserDeck)).
			Where("user_id = ? AND deck_id = ?", row.UserID, row.DeckID).Get(&target)
		if err != nil {
			return err
		}
		if !exists {
			target = usermodel.UserDeck{
				DeckID:     row.DeckID,
				MainFlag:   row.MainFlag,
				DeckName:   row.DeckName,
				UserID:     row.UserID,
				InsertDate: row.InsertDate,
			}
			if target.DeckID <= 0 {
				target.DeckID = 1
			}
			if strings.TrimSpace(target.DeckName) == "" {
				target.DeckName = "队伍A"
			}
			if target.InsertDate <= 0 {
				target.InsertDate = time.Now().Unix()
			}
			if _, err := session.Insert(&target); err != nil {
				return fmt.Errorf("迁移 user_deck_m 失败: %w", err)
			}
			migratedDecks++
		}
		deckMap[row.ID] = migratedDeckRef{TargetDeckID: target.ID, UserID: row.UserID}
	}

	hasUnits, err := db.UserEng.IsTableExist("deck_unit_m")
	if err != nil || !hasUnits {
		return err
	}
	legacyUnits := []legacyDeckUnit{}
	if err := session.Table("deck_unit_m").Find(&legacyUnits); err != nil {
		return fmt.Errorf("读取 deck_unit_m 失败: %w", err)
	}
	migratedUnits := 0
	for _, row := range legacyUnits {
		ref, ok := deckMap[row.UserDeckID]
		if !ok || ref.TargetDeckID <= 0 || ref.UserID <= 0 {
			continue
		}
		exists, err := session.Table(new(usermodel.UserDeckUnit)).
			Where("user_deck_id = ? AND position = ?", ref.TargetDeckID, row.Position).
			Exist()
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		target := usermodel.UserDeckUnit{
			UserDeckID:       ref.TargetDeckID,
			UnitOwningUserID: row.UnitOwningUserID,
			UnitID:           row.UnitID,
			Position:         row.Position,
			Level:            row.Level,
			LevelLimitID:     row.LevelLimitID,
			DisplayRank:      row.DisplayRank,
			Love:             row.Love,
			UnitSkillLevel:   row.UnitSkillLevel,
			IsRankMax:        row.IsRankMax != 0,
			IsLoveMax:        row.IsLoveMax != 0,
			IsLevelMax:       row.IsLevelMax != 0,
			IsSigned:         row.IsSigned != 0,
			BeforeLove:       row.BeforeLove,
			MaxLove:          row.MaxLove,
			UserID:           ref.UserID,
			InsertDate:       row.InsertDate,
		}
		if _, err := session.Insert(&target); err != nil {
			return fmt.Errorf("迁移 deck_unit_m 失败: %w", err)
		}
		migratedUnits++
	}
	if migratedDecks > 0 || migratedUnits > 0 {
		log.Printf("已迁移旧版卡组: user_deck=%d, deck_unit=%d", migratedDecks, migratedUnits)
	}
	return nil
}

func migrateLegacyAccessoryWear(session *xorm.Session) error {
	hasLegacy, err := db.UserEng.IsTableExist("accessory_wear_m")
	if err != nil || !hasLegacy {
		return err
	}
	rows := []legacyAccessoryWear{}
	if err := session.Table("accessory_wear_m").Find(&rows); err != nil {
		return fmt.Errorf("读取 accessory_wear_m 失败: %w", err)
	}
	migrated := 0
	for _, row := range rows {
		if row.UserID <= 0 {
			continue
		}
		exists, err := session.Table(new(usermodel.UserAccessoryWear)).
			Where("user_id = ? AND accessory_owning_user_id = ? AND unit_owning_user_id = ?", row.UserID, row.AccessoryOwningUserID, row.UnitOwningUserID).
			Exist()
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := session.Insert(&usermodel.UserAccessoryWear{
			AccessoryOwningUserID: row.AccessoryOwningUserID,
			UnitOwningUserID:      row.UnitOwningUserID,
			UserID:                row.UserID,
		}); err != nil {
			return fmt.Errorf("迁移 accessory_wear_m 失败: %w", err)
		}
		migrated++
	}
	if migrated > 0 {
		log.Printf("已从 accessory_wear_m 迁移 %d 条装备记录", migrated)
	}
	return nil
}

func migrateLegacySkillEquip(session *xorm.Session) error {
	hasLegacy, err := db.UserEng.IsTableExist("skill_equip_m")
	if err != nil || !hasLegacy {
		return err
	}
	rows := []legacySkillEquip{}
	if err := session.Table("skill_equip_m").Find(&rows); err != nil {
		return fmt.Errorf("读取 skill_equip_m 失败: %w", err)
	}
	migrated := 0
	for _, row := range rows {
		if row.UserID <= 0 {
			continue
		}
		exists, err := session.Table(new(usermodel.UserUnitSkillEquip)).
			Where("user_id = ? AND unit_owning_user_id = ? AND unit_removable_skill_id = ?", row.UserID, row.UnitOwningUserID, row.UnitRemovableSkillID).
			Exist()
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := session.Insert(&usermodel.UserUnitSkillEquip{
			UnitOwningUserID:     row.UnitOwningUserID,
			UnitRemovableSkillID: row.UnitRemovableSkillID,
			UserID:               row.UserID,
		}); err != nil {
			return fmt.Errorf("迁移 skill_equip_m 失败: %w", err)
		}
		migrated++
	}
	if migrated > 0 {
		log.Printf("已从 skill_equip_m 迁移 %d 条技能装备记录", migrated)
	}
	return nil
}

func ensureProfilesForExistingUsers(session *xorm.Session, wrapperPrefs legacyWrapperUserPrefs) error {
	userIDs := []int{}
	if err := session.Table(new(usermodel.Users)).Cols("user_id").Find(&userIDs); err != nil {
		return err
	}
	created := 0
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		exists, err := session.Table(new(usermodel.UserPref)).
			Where("user_id = ?", userID).Exist()
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		unitOwningUserID := 0
		_, err = session.Table(new(usermodel.UserUnit)).
			Where("user_id = ? AND unit_id = ?", userID, 31).
			Cols("unit_owning_user_id").Get(&unitOwningUserID)
		if err != nil {
			return err
		}
		if unitOwningUserID <= 0 {
			_, err = session.Table(new(usermodel.UserUnit)).
				Where("user_id = ?", userID).
				OrderBy("unit_owning_user_id ASC").
				Cols("unit_owning_user_id").Get(&unitOwningUserID)
			if err != nil {
				return err
			}
		}

		pref := usermodel.UserPref{
			UserID:           userID,
			AwardID:          1,
			BackgroundID:     1,
			UnitOwningUserID: unitOwningUserID,
			UserName:         usermodel.DefaultAutoUserName,
			UserLevel:        usermodel.DefaultUserLevel,
			UserDesc:         usermodel.DefaultAutoUserDesc,
			InviteCode:       strconv.Itoa(userID),
			UserExp:          usermodel.DefaultUserExp,
			NextExp:          usermodel.DefaultUserNextExp,
			GameCoin:         usermodel.DefaultUserGameCoin,
			SnsCoin:          usermodel.DefaultUserSnsCoin,
			EnergyMax:        usermodel.DefaultUserEnergyMax,
			OverMaxEnergy:    usermodel.DefaultUserOverMaxEnergy,
			BirthMonth:       usermodel.DefaultBirthMonth,
			BirthDay:         usermodel.DefaultBirthDay,
			ProfileVersion:   usermodel.CurrentUserPrefProfileVersion,
			UpdateTime:       time.Now().Unix(),
		}
		applyLegacyWrapperProfile(&pref, wrapperPrefs)
		pref.ApplyProfileDefaults()
		if _, err := session.Insert(&pref); err != nil {
			return fmt.Errorf("为已有用户 %d 创建缺失的 user_pref 失败: %w", userID, err)
		}
		created++
	}
	if created > 0 {
		log.Printf("已为 %d 个已有账号补建 user_pref", created)
	}
	return nil
}

func MigrateUserPref() {
	session := db.UserEng.NewSession()
	defer session.Close()

	prefList := []usermodel.UserPref{}
	err := session.Table(new(usermodel.UserPref)).
		Where("profile_version < ?", usermodel.CurrentUserPrefProfileVersion).
		Or("profile_version IS NULL").
		Find(&prefList)
	if err != nil {
		log.Fatalln("迁移 user_pref 失败:", err.Error())
	}

	for _, pref := range prefList {
		pref.ApplyProfileDefaults()
		_, err = session.Table(new(usermodel.UserPref)).
			ID(pref.ID).
			Cols(usermodel.UserPrefProfileColumns()...).
			Update(&pref)
		if err != nil {
			log.Fatalln("迁移 user_pref 失败:", err.Error())
		}
	}
}

func ForceAllUsersRelogin() {
	if _, err := db.UserEng.Table(new(usermodel.UserPref)).
		Cols("force_relogin").
		Update(&usermodel.UserPref{ForceRelogin: true}); err != nil {
		log.Fatalln("设置全员重新登录失败:", err.Error())
	}
}

type legacyLevelDBAuthValue struct {
	ClientToken string `json:"client_token"`
	ServerToken string `json:"server_token"`
}

// MigrateLegacyAuthorizeTokens imports the Termux-era LevelDB login session into
// the mainline auth_key table. The old kernel stored userID -> authorize token
// and authorize token -> client/server token JSON in ./data/honoka-chan.db.
// Keeping those tokens is required for an already-installed SIF client to
// continue using its existing session after the server core is upgraded.
func MigrateLegacyAuthorizeTokens() {
	legacyPath := filepath.Clean("./data/honoka-chan.db")
	if _, err := os.Stat(legacyPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("读取旧版登录数据库失败，跳过 token 迁移: %v", err)
		}
		return
	}

	legacyDB, err := leveldb.OpenFile(legacyPath, &opt.Options{ReadOnly: true})
	if err != nil {
		log.Printf("打开旧版登录数据库失败，跳过 token 迁移: %v", err)
		return
	}
	defer legacyDB.Close()

	rows, err := db.UserEng.QueryString(`
		SELECT DISTINCT CAST(user_id AS TEXT) AS user_id
		FROM users
		WHERE user_id IS NOT NULL AND user_id <> 0
		UNION
		SELECT DISTINCT CAST(user_id AS TEXT) AS user_id
		FROM user_pref
		WHERE user_id IS NOT NULL AND user_id <> 0
	`)
	if err != nil {
		log.Printf("读取用户列表失败，跳过旧版 token 迁移: %v", err)
		return
	}

	session := db.UserEng.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		log.Printf("开始旧版 token 迁移事务失败: %v", err)
		return
	}

	migrated := 0
	for _, row := range rows {
		userIDText := strings.TrimSpace(row["user_id"])
		userID, err := strconv.Atoi(userIDText)
		if err != nil || userID <= 0 {
			continue
		}

		tokenBytes, err := legacyDB.Get([]byte(userIDText), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				log.Printf("读取用户 %d 的旧版 token 失败: %v", userID, err)
			}
			continue
		}
		token := strings.TrimSpace(string(tokenBytes))
		if token == "" {
			continue
		}

		authValue := legacyLevelDBAuthValue{}
		if authBytes, err := legacyDB.Get([]byte(token), nil); err == nil {
			if err := json.Unmarshal(authBytes, &authValue); err != nil {
				log.Printf("解析用户 %d 的旧版 token 元数据失败，将只迁移 authorize token: %v", userID, err)
			}
		} else if err != leveldb.ErrNotFound {
			log.Printf("读取用户 %d 的旧版 token 元数据失败: %v", userID, err)
		}

		exists, err := session.Table(new(loginmodel.AuthKey)).
			Where("authorize_token = ? AND user_id = ?", token, userID).
			Exist()
		if err != nil {
			session.Rollback()
			log.Printf("检查用户 %d 的旧版 token 失败: %v", userID, err)
			return
		}
		if !exists {
			_, err = session.Insert(&loginmodel.AuthKey{
				AuthorizeToken: token,
				UserID:         userID,
				ClientToken:    authValue.ClientToken,
				ServerToken:    authValue.ServerToken,
				InsertDate:     time.Now().Format("2006-01-02 15:04:05"),
			})
			if err != nil {
				session.Rollback()
				log.Printf("迁移用户 %d 的旧版 token 失败: %v", userID, err)
				return
			}
			migrated++
		}

		if err := usermodel.ClearUserForceRelogin(session, userID); err != nil {
			session.Rollback()
			log.Printf("清除用户 %d 的强制重新登录标记失败: %v", userID, err)
			return
		}
	}

	if err := session.Commit(); err != nil {
		session.Rollback()
		log.Printf("提交旧版 token 迁移失败: %v", err)
		return
	}
	if migrated > 0 {
		log.Printf("已从旧版 LevelDB 迁移 %d 个登录 token", migrated)
	}
}

func MigrateLegacyUnitTables() {
	if err := db.UserEng.Sync2(new(unitmodel.CommonUnitData), new(usermodel.UserUnitData)); err != nil {
		log.Fatalln("迁移卡片历史数据表失败:", err.Error())
	}
	if err := BackfillCommonUnitExp(); err != nil {
		log.Fatalln("回填卡片经验失败:", err.Error())
	}
}

func BackfillCommonUnitExp() error {
	session := db.UserEng.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return err
	}

	var unitIDs []int
	if err := session.Table(new(unitmodel.CommonUnitData)).Cols("unit_id").Find(&unitIDs); err != nil {
		session.Rollback()
		return err
	}
	if len(unitIDs) == 0 {
		return session.Commit()
	}

	unitRows := []unitmodel.UnitM{}
	if err := db.MainEng.Table(new(unitmodel.UnitM)).
		In("unit_id", unitIDs).
		Cols("unit_id,unit_level_up_pattern_id").
		Find(&unitRows); err != nil {
		session.Rollback()
		return err
	}

	expByPattern := map[int]int{}
	for _, row := range unitRows {
		exp, ok := expByPattern[row.UnitLevelUpPatternId]
		if !ok {
			var err error
			exp, _, err = calculateUnitMaxLevelAndExp(row.UnitLevelUpPatternId)
			if err != nil {
				session.Rollback()
				return err
			}
			expByPattern[row.UnitLevelUpPatternId] = exp
		}

		if _, err := session.Table(new(unitmodel.CommonUnitData)).
			Where("unit_id = ?", row.UnitId).
			Cols("exp").
			Update(&unitmodel.CommonUnitData{Exp: exp}); err != nil {
			session.Rollback()
			return err
		}
	}

	return session.Commit()
}

const legacyDefaultUserID = 377385143

// RepairLegacyUserIDColumns normalizes user IDs before any current Go model is
// scanned. Older kernels used the column name "userid" in tables such as users
// and user_key. After Sync2 adds the current "user_id" column, existing rows are
// NULL. Scanning those NULL values into int fields aborts startup.
//
// This migration is deliberately implemented with raw SQL and schema inspection:
// it never scans a nullable legacy user_id into a Go int. It first copies any
// legacy userid value, then assigns still-unowned rows to the first valid account
// (or the stable default account when no account row exists yet).
func RepairLegacyUserIDColumns() error {
	quoteIdentifier := func(name string) string {
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}

	tableRows, err := db.UserEng.QueryString(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table'
		  AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return err
	}

	tables := make([]string, 0, len(tableRows))
	columnsByTable := make(map[string]map[string]bool, len(tableRows))
	for _, row := range tableRows {
		table := strings.TrimSpace(row["name"])
		if table == "" {
			continue
		}
		tables = append(tables, table)

		columnRows, err := db.UserEng.QueryString(
			"PRAGMA table_info(" + quoteIdentifier(table) + ")",
		)
		if err != nil {
			return fmt.Errorf("读取表 %s 的结构失败: %w", table, err)
		}

		columns := make(map[string]bool, len(columnRows))
		for _, columnRow := range columnRows {
			columnName := strings.ToLower(strings.TrimSpace(columnRow["name"]))
			if columnName != "" {
				columns[columnName] = true
			}
		}
		columnsByTable[table] = columns
	}

	// Copy the old users.userid value into the new users.user_id column before
	// selecting an owner account. This is the immediate cause of the reported
	// "converting NULL to int is unsupported" startup failure.
	if columns := columnsByTable["users"]; columns["user_id"] && columns["userid"] {
		if _, err := db.UserEng.Exec(`
			UPDATE "users"
			SET "user_id" = CAST("userid" AS INTEGER)
			WHERE ("user_id" IS NULL OR "user_id" = 0)
			  AND "userid" IS NOT NULL
			  AND CAST("userid" AS INTEGER) <> 0
		`); err != nil {
			return fmt.Errorf("迁移 users.userid 到 users.user_id 失败: %w", err)
		}
	}

	// If a legacy users row still has no ID, give it a deterministic unique ID.
	// A one-row database receives the stable default ID 377385143.
	if columns := columnsByTable["users"]; columns["user_id"] {
		idExpression := "rowid"
		if columns["id"] {
			idExpression = `CASE WHEN "id" IS NULL OR "id" <= 0 THEN rowid ELSE "id" END`
		}
		query := fmt.Sprintf(`
			UPDATE "users"
			SET "user_id" = ? + (%s - 1)
			WHERE "user_id" IS NULL OR "user_id" = 0
		`, idExpression)
		if _, err := db.UserEng.Exec(query, legacyDefaultUserID); err != nil {
			return fmt.Errorf("补全 users.user_id 失败: %w", err)
		}
	}

	ownerUserID := legacyDefaultUserID
	if columns := columnsByTable["users"]; columns["user_id"] {
		orderBy := "rowid"
		if columns["id"] {
			orderBy = `"id"`
		}
		ownerRows, err := db.UserEng.QueryString(fmt.Sprintf(`
			SELECT CAST("user_id" AS TEXT) AS resolved_user_id
			FROM "users"
			WHERE "user_id" IS NOT NULL AND "user_id" <> 0
			ORDER BY %s
			LIMIT 1
		`, orderBy))
		if err != nil {
			return fmt.Errorf("读取迁移目标用户失败: %w", err)
		}
		if len(ownerRows) > 0 {
			if value := strings.TrimSpace(ownerRows[0]["resolved_user_id"]); value != "" {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("解析迁移目标用户 %q 失败: %w", value, err)
				}
				if parsed > 0 {
					ownerUserID = parsed
				}
			}
		}
	}

	for _, table := range tables {
		columns := columnsByTable[table]
		if !columns["user_id"] || table == "users" {
			continue
		}

		quotedTable := quoteIdentifier(table)
		if columns["userid"] {
			query := fmt.Sprintf(`
				UPDATE %s
				SET "user_id" = CAST("userid" AS INTEGER)
				WHERE ("user_id" IS NULL OR "user_id" = 0)
				  AND "userid" IS NOT NULL
				  AND CAST("userid" AS INTEGER) <> 0
			`, quotedTable)
			if _, err := db.UserEng.Exec(query); err != nil {
				return fmt.Errorf("迁移 %s.userid 到 user_id 失败: %w", table, err)
			}
		}

		query := fmt.Sprintf(`
			UPDATE %s
			SET "user_id" = ?
			WHERE "user_id" IS NULL OR "user_id" = 0
		`, quotedTable)
		result, err := db.UserEng.Exec(query, ownerUserID)
		if err != nil {
			return fmt.Errorf("补全 %s.user_id 失败: %w", table, err)
		}
		if affected, err := result.RowsAffected(); err == nil && affected > 0 {
			log.Printf("迁移 %s: 已补全 %d 条旧版 user_id，归属用户 %d", table, affected, ownerUserID)
		}
	}

	return nil
}

type accessoryOwningMapRow struct {
	AccessoryOwningUserID int `xorm:"accessory_owning_user_id"`
	AccessoryID           int `xorm:"accessory_id"`
}

func MigrateUserAccessories() {
	session := db.UserEng.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		log.Fatalln("迁移 user_accessory 失败:", err.Error())
	}

	userIDs := []int{}
	if err := session.Table(new(usermodel.Users)).Cols("user_id").Find(&userIDs); err != nil {
		session.Rollback()
		log.Fatalln("迁移 user_accessory 失败:", err.Error())
	}

	templateAccessoryIDs, err := usermodel.LoadAccessoryTemplateIDs(db.MainEng)
	if err != nil {
		session.Rollback()
		log.Fatalln("迁移 user_accessory 失败:", err.Error())
	}

	legacyAccessoryMap := map[int]int{}
	hasLegacyTable, err := db.MainEng.IsTableExist("common_accessory_m")
	if err != nil {
		session.Rollback()
		log.Fatalln("迁移 user_accessory 失败:", err.Error())
	}
	if hasLegacyTable {
		legacyRows := []accessoryOwningMapRow{}
		if err := db.MainEng.Table("common_accessory_m").
			Cols("accessory_owning_user_id,accessory_id").
			Find(&legacyRows); err != nil {
			session.Rollback()
			log.Fatalln("迁移 user_accessory 失败:", err.Error())
		}
		for _, row := range legacyRows {
			legacyAccessoryMap[row.AccessoryOwningUserID] = row.AccessoryID
		}
	}

	wearRows := []usermodel.UserAccessoryWear{}
	if err := session.Table(new(usermodel.UserAccessoryWear)).Find(&wearRows); err != nil {
		session.Rollback()
		log.Fatalln("迁移 user_accessory 失败:", err.Error())
	}

	existingAccessories := []usermodel.UserAccessory{}
	if err := session.Table(new(usermodel.UserAccessory)).Find(&existingAccessories); err != nil {
		session.Rollback()
		log.Fatalln("迁移 user_accessory 失败:", err.Error())
	}

	currentAccessoryMap := make(map[int]map[int]int)
	for _, row := range existingAccessories {
		if currentAccessoryMap[row.UserID] == nil {
			currentAccessoryMap[row.UserID] = map[int]int{}
		}
		currentAccessoryMap[row.UserID][row.AccessoryOwningUserID] = row.AccessoryID
	}

	requiredCountsByUser := make(map[int]map[int]int, len(userIDs))
	for _, row := range wearRows {
		accessoryID, ok := resolveAccessoryIDForWear(currentAccessoryMap, legacyAccessoryMap, templateAccessoryIDs, row.UserID, row.AccessoryOwningUserID)
		if !ok {
			continue
		}
		if requiredCountsByUser[row.UserID] == nil {
			requiredCountsByUser[row.UserID] = map[int]int{}
		}
		requiredCountsByUser[row.UserID][accessoryID]++
	}

	for _, userID := range userIDs {
		if err := usermodel.EnsureUserAccessories(session, db.MainEng, userID, requiredCountsByUser[userID]); err != nil {
			session.Rollback()
			log.Fatalln("迁移 user_accessory 失败:", err.Error())
		}
	}

	updatedAccessories := []usermodel.UserAccessory{}
	if err := session.Table(new(usermodel.UserAccessory)).
		OrderBy("user_id ASC, accessory_id ASC, accessory_owning_user_id ASC").
		Find(&updatedAccessories); err != nil {
		session.Rollback()
		log.Fatalln("迁移 user_accessory 失败:", err.Error())
	}

	currentAccessoryMap = make(map[int]map[int]int)
	availableOwningIDs := make(map[int]map[int][]int)
	for _, row := range updatedAccessories {
		if currentAccessoryMap[row.UserID] == nil {
			currentAccessoryMap[row.UserID] = map[int]int{}
		}
		if availableOwningIDs[row.UserID] == nil {
			availableOwningIDs[row.UserID] = map[int][]int{}
		}
		currentAccessoryMap[row.UserID][row.AccessoryOwningUserID] = row.AccessoryID
		availableOwningIDs[row.UserID][row.AccessoryID] = append(availableOwningIDs[row.UserID][row.AccessoryID], row.AccessoryOwningUserID)
	}

	usedOwningIDs := make(map[int]map[int]struct{})
	for _, row := range wearRows {
		if currentAccessoryMap[row.UserID] == nil {
			continue
		}
		if _, ok := currentAccessoryMap[row.UserID][row.AccessoryOwningUserID]; !ok {
			continue
		}
		if usedOwningIDs[row.UserID] == nil {
			usedOwningIDs[row.UserID] = map[int]struct{}{}
		}
		usedOwningIDs[row.UserID][row.AccessoryOwningUserID] = struct{}{}
	}

	for i := range wearRows {
		row := wearRows[i]
		if row.AccessoryOwningUserID <= 0 {
			continue
		}
		if currentAccessoryMap[row.UserID] != nil {
			if _, ok := currentAccessoryMap[row.UserID][row.AccessoryOwningUserID]; ok {
				continue
			}
		}

		accessoryID, ok := resolveAccessoryIDForWear(currentAccessoryMap, legacyAccessoryMap, templateAccessoryIDs, row.UserID, row.AccessoryOwningUserID)
		if !ok {
			continue
		}

		candidates := availableOwningIDs[row.UserID][accessoryID]
		if len(candidates) == 0 {
			continue
		}

		if usedOwningIDs[row.UserID] == nil {
			usedOwningIDs[row.UserID] = map[int]struct{}{}
		}

		newOwningID := 0
		for _, candidate := range candidates {
			if _, used := usedOwningIDs[row.UserID][candidate]; used {
				continue
			}
			newOwningID = candidate
			break
		}
		if newOwningID == 0 {
			newOwningID = candidates[0]
		}

		if _, err := session.Table(new(usermodel.UserAccessoryWear)).
			ID(row.ID).
			Cols("accessory_owning_user_id").
			Update(&usermodel.UserAccessoryWear{AccessoryOwningUserID: newOwningID}); err != nil {
			session.Rollback()
			log.Fatalln("迁移 user_accessory 失败:", err.Error())
		}
		wearRows[i].AccessoryOwningUserID = newOwningID
		usedOwningIDs[row.UserID][newOwningID] = struct{}{}
	}

	if err := session.Commit(); err != nil {
		session.Rollback()
		log.Fatalln("迁移 user_accessory 失败:", err.Error())
	}
}

func resolveAccessoryIDForWear(currentAccessoryMap map[int]map[int]int, legacyAccessoryMap map[int]int, accessoryIDs []int, userID int, accessoryOwningUserID int) (int, bool) {
	if accessoryOwningUserID <= 0 {
		return 0, false
	}

	if currentAccessoryMap[userID] != nil {
		if accessoryID, ok := currentAccessoryMap[userID][accessoryOwningUserID]; ok {
			return accessoryID, true
		}
	}

	if accessoryID, ok := legacyAccessoryMap[accessoryOwningUserID]; ok {
		return accessoryID, true
	}

	return legacyAccessoryOwningUserIDToAccessoryID(accessoryIDs, accessoryOwningUserID)
}

func legacyAccessoryOwningUserIDToAccessoryID(accessoryIDs []int, accessoryOwningUserID int) (int, bool) {
	if accessoryOwningUserID <= 0 {
		return 0, false
	}

	index := (accessoryOwningUserID - 1) / 9
	if index < 0 || index >= len(accessoryIDs) {
		return 0, false
	}

	return accessoryIDs[index], true
}

type liveGoalRewardMigrationRow struct {
	LiveGoalRewardID int `xorm:"live_goal_reward_id"`
	LiveDifficultyID int `xorm:"live_difficulty_id"`
	LiveGoalType     int `xorm:"live_goal_type"`
	Rank             int `xorm:"rank"`
}

type liveGoalInfoMigrationRow struct {
	LiveDifficultyID int `xorm:"live_difficulty_id"`
	CRankScore       int `xorm:"c_rank_score"`
	BRankScore       int `xorm:"b_rank_score"`
	ARankScore       int `xorm:"a_rank_score"`
	SRankScore       int `xorm:"s_rank_score"`
	CRankCombo       int `xorm:"c_rank_combo"`
	BRankCombo       int `xorm:"b_rank_combo"`
	ARankCombo       int `xorm:"a_rank_combo"`
	SRankCombo       int `xorm:"s_rank_combo"`
	CRankComplete    int `xorm:"c_rank_complete"`
	BRankComplete    int `xorm:"b_rank_complete"`
	ARankComplete    int `xorm:"a_rank_complete"`
	SRankComplete    int `xorm:"s_rank_complete"`
}

func liveRankForMigration(value int, cRank int, bRank int, aRank int, sRank int) int {
	switch {
	case value >= sRank:
		return 1
	case value >= aRank:
		return 2
	case value >= bRank:
		return 3
	case value >= cRank:
		return 4
	default:
		return 5
	}
}

func MigrateUserLiveData() {
	session := db.UserEng.NewSession()
	defer session.Close()

	recordRows := []usermodel.UserLiveRecord{}
	if err := session.Table(new(usermodel.UserLiveRecord)).Find(&recordRows); err != nil {
		log.Fatalln("迁移 user_live_status 失败:", err.Error())
	}

	existingStatusRows := []usermodel.UserLiveStatus{}
	if err := session.Table(new(usermodel.UserLiveStatus)).Find(&existingStatusRows); err != nil {
		log.Fatalln("迁移 user_live_status 失败:", err.Error())
	}

	type liveStatusKey struct {
		UserID           int
		LiveDifficultyID int
	}

	statusMap := make(map[liveStatusKey]usermodel.UserLiveStatus, len(existingStatusRows))
	for _, row := range existingStatusRows {
		statusMap[liveStatusKey{UserID: row.UserID, LiveDifficultyID: row.LiveDifficultyID}] = row
	}

	now := time.Now().Unix()
	for _, row := range recordRows {
		key := liveStatusKey{UserID: row.UserID, LiveDifficultyID: row.LiveDifficultyID}
		status, ok := statusMap[key]
		if !ok {
			status = usermodel.UserLiveStatus{
				UserID:           row.UserID,
				LiveDifficultyID: row.LiveDifficultyID,
				HiScore:          row.TotalScore,
				HiComboCount:     row.MaxCombo,
				ClearCnt:         1,
				InsertDate:       now,
				UpdateDate:       now,
			}
			if _, err := session.Insert(&status); err != nil {
				log.Fatalln("迁移 user_live_status 失败:", err.Error())
			}
			statusMap[key] = status
			continue
		}

		updated := false
		if row.TotalScore > status.HiScore {
			status.HiScore = row.TotalScore
			updated = true
		}
		if row.MaxCombo > status.HiComboCount {
			status.HiComboCount = row.MaxCombo
			updated = true
		}
		if status.ClearCnt <= 0 {
			status.ClearCnt = 1
			updated = true
		}
		if updated {
			status.UpdateDate = now
			if _, err := session.Table(new(usermodel.UserLiveStatus)).
				Where("user_id = ? AND live_difficulty_id = ?", status.UserID, status.LiveDifficultyID).
				Cols("hi_score", "hi_combo_count", "clear_cnt", "update_date").
				Update(&status); err != nil {
				log.Fatalln("迁移 user_live_status 失败:", err.Error())
			}
			statusMap[key] = status
		}
	}

	liveGoalInfoMap := map[int]liveGoalInfoMigrationRow{}
	loadLiveGoalInfoRows := func(table string) {
		rows := []liveGoalInfoMigrationRow{}
		err := db.MainEng.Table(table).Alias("live").
			Join("LEFT", "live_setting_m setting", "live.live_setting_id = setting.live_setting_id").
			Select(`
				live.live_difficulty_id,
				setting.c_rank_score,
				setting.b_rank_score,
				setting.a_rank_score,
				setting.s_rank_score,
				setting.c_rank_combo,
				setting.b_rank_combo,
				setting.a_rank_combo,
				setting.s_rank_combo,
				live.c_rank_complete,
				live.b_rank_complete,
				live.a_rank_complete,
				live.s_rank_complete
			`).
			Find(&rows)
		if err != nil {
			log.Fatalln("迁移 user_live_goal 失败:", err.Error())
		}
		for _, row := range rows {
			liveGoalInfoMap[row.LiveDifficultyID] = row
		}
	}
	loadLiveGoalInfoRows("normal_live_m")
	loadLiveGoalInfoRows("special_live_m")

	goalRewardRows := []liveGoalRewardMigrationRow{}
	if err := db.MainEng.Table("live_goal_reward_m").
		OrderBy("live_difficulty_id ASC, live_goal_type ASC, rank ASC, live_goal_reward_id ASC").
		Find(&goalRewardRows); err != nil {
		log.Fatalln("迁移 user_live_goal 失败:", err.Error())
	}

	goalRewardMap := map[int][]liveGoalRewardMigrationRow{}
	for _, row := range goalRewardRows {
		goalRewardMap[row.LiveDifficultyID] = append(goalRewardMap[row.LiveDifficultyID], row)
	}

	existingGoalRows := []usermodel.UserLiveGoal{}
	if err := session.Table(new(usermodel.UserLiveGoal)).
		Where("live_goal_reward_id > 0").
		Find(&existingGoalRows); err != nil {
		log.Fatalln("迁移 user_live_goal 失败:", err.Error())
	}

	existingGoalMap := make(map[liveStatusKey]map[int]struct{})
	for _, row := range existingGoalRows {
		key := liveStatusKey{UserID: row.UserID, LiveDifficultyID: row.LiveDifficultyID}
		if existingGoalMap[key] == nil {
			existingGoalMap[key] = map[int]struct{}{}
		}
		existingGoalMap[key][row.LiveGoalRewardID] = struct{}{}
	}

	statusKeys := make([]liveStatusKey, 0, len(statusMap))
	for key := range statusMap {
		statusKeys = append(statusKeys, key)
	}
	sort.Slice(statusKeys, func(i, j int) bool {
		if statusKeys[i].UserID == statusKeys[j].UserID {
			return statusKeys[i].LiveDifficultyID < statusKeys[j].LiveDifficultyID
		}
		return statusKeys[i].UserID < statusKeys[j].UserID
	})

	for _, key := range statusKeys {
		status := statusMap[key]
		liveInfo, ok := liveGoalInfoMap[key.LiveDifficultyID]
		if !ok {
			continue
		}

		scoreRank := liveRankForMigration(status.HiScore, liveInfo.CRankScore, liveInfo.BRankScore, liveInfo.ARankScore, liveInfo.SRankScore)
		comboRank := liveRankForMigration(status.HiComboCount, liveInfo.CRankCombo, liveInfo.BRankCombo, liveInfo.ARankCombo, liveInfo.SRankCombo)
		clearRank := liveRankForMigration(status.ClearCnt, liveInfo.CRankComplete, liveInfo.BRankComplete, liveInfo.ARankComplete, liveInfo.SRankComplete)

		for _, reward := range goalRewardMap[key.LiveDifficultyID] {
			achieved := false
			switch reward.LiveGoalType {
			case 1:
				achieved = scoreRank <= reward.Rank
			case 2:
				achieved = comboRank <= reward.Rank
			case 3:
				achieved = clearRank <= reward.Rank
			}
			if !achieved {
				continue
			}

			if existingGoalMap[key] != nil {
				if _, ok := existingGoalMap[key][reward.LiveGoalRewardID]; ok {
					continue
				}
			}

			if _, err := session.Insert(&usermodel.UserLiveGoal{
				UserID:           key.UserID,
				LiveDifficultyID: key.LiveDifficultyID,
				LiveGoalRewardID: reward.LiveGoalRewardID,
				GoalType:         constant.LiveGoalType(reward.LiveGoalType),
				Rank:             reward.Rank,
				CompletedAt:      now,
			}); err != nil {
				log.Fatalln("迁移 user_live_goal 失败:", err.Error())
			}
			if existingGoalMap[key] == nil {
				existingGoalMap[key] = map[int]struct{}{}
			}
			existingGoalMap[key][reward.LiveGoalRewardID] = struct{}{}
		}
	}
}

func LoadUnitData() {
	userEng = db.UserEng.NewSession()
	defer userEng.Close()

	err := userEng.Begin()
	CheckErr(err)

	commonUnitExist, err := userEng.IsTableExist(new(unitmodel.CommonUnitData))
	CheckErr(err)
	userUnitExist, err := userEng.IsTableExist(new(usermodel.UserUnitData))
	CheckErr(err)

	needInit := !commonUnitExist || !userUnitExist
	if !needInit {
		commonUnitCount, err := userEng.Table(new(unitmodel.CommonUnitData)).Count()
		CheckErr(err)
		userUnitCount, err := userEng.Table(new(usermodel.UserUnitData)).Count()
		CheckErr(err)
		needInit = commonUnitCount == 0 && userUnitCount == 0
	}

	if needInit {
		log.Println("卡片数据不存在，正在同步...")

		userEng.DropTable(new(unitmodel.CommonUnitData))
		userEng.CreateTable(new(unitmodel.CommonUnitData))
		userEng.DropTable(new(usermodel.UserUnitData))
		userEng.CreateTable(new(usermodel.UserUnitData))

		var unitData []unitmodel.UnitM
		err = db.MainEng.Table(new(unitmodel.UnitM)).OrderBy("unit_id ASC").Find(&unitData)
		CheckErr(err)

		checked := false
		for _, u := range unitData {
			sumExp, unitMaxLevel, err := calculateUnitMaxLevelAndExp(u.UnitLevelUpPatternId)
			CheckErr(err)

			// 计算突破前的属性
			var smileMax, pureMax, coolMax int
			smileMax = u.SmileMax
			pureMax = u.PureMax
			coolMax = u.CoolMax

			if unitMaxLevel == 350 {
				// 计算突破后的经验总和
				// 计算突破后的属性
				smileMax += 6000
				pureMax += 6000
				coolMax += 6000
			}

			// 计算绊值、技能等级、技能经验
			var maxLove, skillLevel, skillExp, removableSkillCapacity, levelLimitID int
			switch u.Rarity {
			case 1:
				maxLove = 50
				skillExp = 0
				skillLevel = 0
				removableSkillCapacity = 0
				levelLimitID = 0
			case 2:
				maxLove = 200
				skillExp = 490
				skillLevel = 8
				removableSkillCapacity = 1
				levelLimitID = 0
			case 3:
				maxLove = 500
				skillExp = 4900
				skillLevel = 8
				removableSkillCapacity = 2
				levelLimitID = 0
			case 4:
				maxLove = 1000
				skillExp = 29900
				skillLevel = 8
				removableSkillCapacity = 8
				levelLimitID = 1
			case 5:
				maxLove = 750
				skillExp = 12700
				skillLevel = 8
				removableSkillCapacity = 3
				levelLimitID = 0
			}

			// 针对技能卡等应援卡片
			if smileMax == 1 {
				maxLove = 0
				skillExp = 0
				skillLevel = 0
				removableSkillCapacity = 0
			}

			// 检查是否签名卡
			var isSigned bool
			exist, err := db.MainEng.Table("unit_sign_asset_m").Where("unit_id = ?", u.UnitId).Exist()
			CheckErr(err)
			if exist {
				isSigned = true
			}

			// 生成公共卡片
			unitCommon := unitmodel.CommonUnitData{
				UnitNumber:                  u.UnitNumber,
				UnitID:                      u.UnitId,
				UnitTypeID:                  u.UnitTypeId,
				Name:                        *u.NameEn,
				Eponym:                      u.EponymEn,
				Rarity:                      u.Rarity,
				Attribute:                   u.AttributeId,
				Smile:                       smileMax,
				Cute:                        pureMax,
				Cool:                        coolMax,
				Exp:                         sumExp,
				Level:                       unitMaxLevel,
				MaxLevel:                    unitMaxLevel,
				LevelLimitID:                levelLimitID,
				Rank:                        u.RankMin,
				MaxRank:                     u.RankMax,
				Love:                        maxLove,
				MaxLove:                     maxLove,
				UnitSkillExp:                skillExp,
				UnitSkillLevel:              skillLevel,
				MaxHp:                       u.HpMax,
				UnitRemovableSkillCapacity:  removableSkillCapacity,
				IsRankMax:                   true,
				IsLoveMax:                   true,
				IsLevelMax:                  true,
				IsSigned:                    isSigned,
				IsSkillLevelMax:             true,
				IsRemovableSkillCapacityMax: true,
				InsertDate:                  time.Now().Unix(),
			}

			_, err = userEng.Insert(&unitCommon)
			CheckErr(err)

			var userID []int
			err = db.UserEng.Table(new(usermodel.Users)).Cols("user_id").Find(&userID)
			CheckErr(err)

			for _, id := range userID {
				userUnit := usermodel.UserUnitData{
					UnitID:       u.UnitId,
					FavoriteFlag: false,
					DisplayRank:  u.RankMax,
					UserID:       id,
					InsertDate:   time.Now().Unix(),
				}

				// 检查表里是否已经有数据
				if !checked {
					ct, err := userEng.Table(new(usermodel.UserUnitData)).Count()
					CheckErr(err)

					if ct == 0 {
						userUnit.UnitOwningUserID = 38383
					}

					checked = true
				}

				_, err = userEng.Insert(&userUnit)
				CheckErr(err)
			}
		}

		err = userEng.Commit()
		CheckErr(err)

		log.Println("同步完成！")
	}
}

func calculateUnitMaxLevelAndExp(unitLevelUpPatternID int) (sumExp int, unitMaxLevel int, err error) {
	var nextExp int
	_, err = db.MainEng.Table("unit_level_up_pattern_m").
		Where("unit_level_up_pattern_id = ?", unitLevelUpPatternID).
		Select("MAX(unit_level),next_exp").
		Get(&unitMaxLevel, &nextExp)
	if err != nil {
		return 0, 0, err
	}

	_, err = db.MainEng.Table("unit_level_up_pattern_m").
		Where("unit_level_up_pattern_id = ?", unitLevelUpPatternID).
		Where("unit_level = ?", unitMaxLevel-1).
		Cols("next_exp").
		Get(&sumExp)
	if err != nil {
		return 0, 0, err
	}

	if nextExp != 0 {
		_, err = db.MainEng.Table("unit_level_limit_pattern_m").
			Where("unit_level_limit_id = 1 AND unit_level = 349").
			Cols("next_exp").
			Get(&sumExp)
		if err != nil {
			return 0, 0, err
		}
		unitMaxLevel = 350
	}

	return sumExp, unitMaxLevel, nil
}

func CheckErr(err error) {
	if err != nil {
		userEng.Rollback()
		log.Fatalln("同步失败:", err.Error())
	}
}
