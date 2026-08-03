package startup

import (
	"fmt"
	unitmodel "honoka-chan/internal/model/unit"
	usermodel "honoka-chan/internal/model/user"
	"honoka-chan/pkg/db"
	"log"
	"sort"
	"strings"
	"time"

	"xorm.io/xorm"
)

// legacyUnitPreference contains only the old user_unit_m columns that are still
// meaningful in the current user_unit_data model. The old table stored a full
// per-user copy of card data, while the current core stores immutable card data
// in common_unit_data and only ownership/favourite/display state per user.
type legacyUnitPreference struct {
	UnitOwningUserID int `xorm:"unit_owning_user_id"`
	UserID           int `xorm:"user_id"`
	UnitID           int `xorm:"unit_id"`
	FavoriteFlag     int `xorm:"favorite_flag"`
	DisplayRank      int `xorm:"display_rank"`
}

type resolvedDeckUnit struct {
	UserUnit usermodel.UserUnitData
	Common   unitmodel.CommonUnitData
}

var canonicalDefaultDeckUnitIDs = []int{3465, 3466, 3467, 3468, 3469, 3470, 3471, 3472, 3473}

// ReconcileLegacyUserState repairs the semantic relationships that cannot be
// preserved by merely copying old Termux tables to similarly named mainline
// tables. In particular, the current API reads user_unit_data joined with
// common_unit_data, while old decks refer to ownership IDs from common_unit_m /
// user_unit_m. A deck that references an ownership ID absent from unitAll can
// make the native client abort while consuming /main.php/api.
//
// The routine is idempotent. Valid current-mainline rows are kept; only missing
// ownership mappings, stale centre-unit references, and invalid deck rows are
// created or repaired.
func ReconcileLegacyUserState() {
	session := db.UserEng.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		log.Fatalln("开始旧版用户状态校验失败:", err.Error())
	}

	if err := reconcileAllLegacyUsers(session); err != nil {
		_ = session.Rollback()
		log.Fatalln("校验旧版用户状态失败:", err.Error())
	}

	if err := session.Commit(); err != nil {
		_ = session.Rollback()
		log.Fatalln("提交旧版用户状态校验失败:", err.Error())
	}
}

func reconcileAllLegacyUsers(session *xorm.Session) error {
	userIDs := []int{}
	if err := session.Table(new(usermodel.Users)).
		Where("user_id IS NOT NULL AND user_id > 0").
		Cols("user_id").
		OrderBy("user_id ASC").
		Find(&userIDs); err != nil {
		return fmt.Errorf("读取用户列表失败: %w", err)
	}

	if len(userIDs) == 0 {
		return nil
	}

	commonUnits := []unitmodel.CommonUnitData{}
	if err := session.Table(new(unitmodel.CommonUnitData)).
		OrderBy("unit_id ASC").
		Find(&commonUnits); err != nil {
		return fmt.Errorf("读取 common_unit_data 失败: %w", err)
	}
	if len(commonUnits) == 0 {
		return fmt.Errorf("common_unit_data 为空，无法修复用户卡片状态")
	}

	commonByUnitID := make(map[int]unitmodel.CommonUnitData, len(commonUnits))
	for _, unit := range commonUnits {
		commonByUnitID[unit.UnitID] = unit
	}

	occupiedOwningIDs := map[int]struct{}{}
	allOwningIDs := []int{}
	if err := session.Table(new(usermodel.UserUnitData)).
		Cols("unit_owning_user_id").
		Find(&allOwningIDs); err != nil {
		return fmt.Errorf("读取现有 unit_owning_user_id 失败: %w", err)
	}
	for _, id := range allOwningIDs {
		if id > 0 {
			occupiedOwningIDs[id] = struct{}{}
		}
	}

	for _, userID := range userIDs {
		legacyPrefs, err := loadLegacyUnitPreferences(session, userID)
		if err != nil {
			return err
		}

		unitByID, unitByOwningID, inserted, updated, err := ensureUserUnitMappings(
			session,
			userID,
			commonUnits,
			legacyPrefs,
			occupiedOwningIDs,
		)
		if err != nil {
			return err
		}

		centerChanged, err := repairUserCenterUnit(session, userID, unitByID, unitByOwningID)
		if err != nil {
			return err
		}

		deckRepaired, err := repairUserDecks(session, userID, commonByUnitID, unitByID, unitByOwningID)
		if err != nil {
			return err
		}

		if inserted > 0 || updated > 0 || centerChanged || deckRepaired > 0 {
			log.Printf(
				"用户 %d 状态校验完成: 新增卡片映射=%d, 更新卡片状态=%d, 修复中心成员=%t, 修复卡组=%d",
				userID,
				inserted,
				updated,
				centerChanged,
				deckRepaired,
			)
		}
	}

	return nil
}

func loadLegacyUnitPreferences(session *xorm.Session, userID int) (map[int]legacyUnitPreference, error) {
	hasLegacy, err := db.UserEng.IsTableExist("user_unit_m")
	if err != nil || !hasLegacy {
		return map[int]legacyUnitPreference{}, err
	}

	rows := []legacyUnitPreference{}
	if err := session.Table("user_unit_m").
		Where("user_id = ?", userID).
		Cols("unit_owning_user_id,user_id,unit_id,favorite_flag,display_rank").
		Find(&rows); err != nil {
		return nil, fmt.Errorf("读取用户 %d 的 user_unit_m 失败: %w", userID, err)
	}

	result := make(map[int]legacyUnitPreference, len(rows))
	for _, row := range rows {
		if row.UnitID > 0 {
			result[row.UnitID] = row
		}
	}
	return result, nil
}

func ensureUserUnitMappings(
	session *xorm.Session,
	userID int,
	commonUnits []unitmodel.CommonUnitData,
	legacyPrefs map[int]legacyUnitPreference,
	occupiedOwningIDs map[int]struct{},
) (map[int]usermodel.UserUnitData, map[int]usermodel.UserUnitData, int, int, error) {
	existing := []usermodel.UserUnitData{}
	if err := session.Table(new(usermodel.UserUnitData)).
		Where("user_id = ?", userID).
		OrderBy("unit_owning_user_id ASC").
		Find(&existing); err != nil {
		return nil, nil, 0, 0, fmt.Errorf("读取用户 %d 的 user_unit_data 失败: %w", userID, err)
	}

	byUnitID := make(map[int]usermodel.UserUnitData, len(existing))
	byOwningID := make(map[int]usermodel.UserUnitData, len(existing))
	for _, row := range existing {
		if row.UnitID > 0 {
			byUnitID[row.UnitID] = row
		}
		if row.UnitOwningUserID > 0 {
			byOwningID[row.UnitOwningUserID] = row
			occupiedOwningIDs[row.UnitOwningUserID] = struct{}{}
		}
	}

	inserted := 0
	updated := 0
	for _, common := range commonUnits {
		row, exists := byUnitID[common.UnitID]
		legacy, hasLegacy := legacyPrefs[common.UnitID]
		if exists {
			newFavorite := row.FavoriteFlag
			newDisplayRank := row.DisplayRank
			if hasLegacy {
				newFavorite = legacy.FavoriteFlag != 0
				if legacy.DisplayRank > 0 {
					newDisplayRank = legacy.DisplayRank
				}
			}
			if newDisplayRank <= 0 {
				newDisplayRank = common.MaxRank
			}
			if newFavorite != row.FavoriteFlag || newDisplayRank != row.DisplayRank {
				row.FavoriteFlag = newFavorite
				row.DisplayRank = newDisplayRank
				if _, err := session.Table(new(usermodel.UserUnitData)).
					Where("user_id = ? AND unit_owning_user_id = ?", userID, row.UnitOwningUserID).
					Cols("favorite_flag", "display_rank").
					Update(&row); err != nil {
					return nil, nil, inserted, updated, fmt.Errorf("更新用户 %d 卡片 %d 状态失败: %w", userID, common.UnitID, err)
				}
				updated++
				byUnitID[common.UnitID] = row
				byOwningID[row.UnitOwningUserID] = row
			}
			continue
		}

		newRow := usermodel.UserUnitData{
			UnitID:       common.UnitID,
			FavoriteFlag: false,
			DisplayRank:  common.MaxRank,
			UserID:       userID,
			InsertDate:   time.Now().Unix(),
		}
		if hasLegacy {
			newRow.FavoriteFlag = legacy.FavoriteFlag != 0
			if legacy.DisplayRank > 0 {
				newRow.DisplayRank = legacy.DisplayRank
			}
			if legacy.UnitOwningUserID > 0 {
				if _, used := occupiedOwningIDs[legacy.UnitOwningUserID]; !used {
					newRow.UnitOwningUserID = legacy.UnitOwningUserID
				}
			}
		}

		if _, err := session.Insert(&newRow); err != nil {
			return nil, nil, inserted, updated, fmt.Errorf("为用户 %d 创建 unit_id=%d 的卡片映射失败: %w", userID, common.UnitID, err)
		}
		if newRow.UnitOwningUserID <= 0 {
			return nil, nil, inserted, updated, fmt.Errorf("为用户 %d 创建 unit_id=%d 后未获得 unit_owning_user_id", userID, common.UnitID)
		}
		occupiedOwningIDs[newRow.UnitOwningUserID] = struct{}{}
		byUnitID[newRow.UnitID] = newRow
		byOwningID[newRow.UnitOwningUserID] = newRow
		inserted++
	}

	return byUnitID, byOwningID, inserted, updated, nil
}

func repairUserCenterUnit(
	session *xorm.Session,
	userID int,
	unitByID map[int]usermodel.UserUnitData,
	unitByOwningID map[int]usermodel.UserUnitData,
) (bool, error) {
	pref := usermodel.UserPref{}
	has, err := session.Table(new(usermodel.UserPref)).Where("user_id = ?", userID).Get(&pref)
	if err != nil {
		return false, fmt.Errorf("读取用户 %d 的 user_pref 失败: %w", userID, err)
	}
	if !has {
		return false, nil
	}
	if pref.UnitOwningUserID > 0 {
		if _, exists := unitByOwningID[pref.UnitOwningUserID]; exists {
			return false, nil
		}
	}

	replacement := usermodel.UserUnitData{}
	for _, preferredUnitID := range []int{338, 31} {
		if row, exists := unitByID[preferredUnitID]; exists {
			replacement = row
			break
		}
	}
	if replacement.UnitOwningUserID <= 0 {
		rows := make([]usermodel.UserUnitData, 0, len(unitByID))
		for _, row := range unitByID {
			rows = append(rows, row)
		}
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].UnitOwningUserID < rows[j].UnitOwningUserID
		})
		if len(rows) > 0 {
			replacement = rows[0]
		}
	}
	if replacement.UnitOwningUserID <= 0 {
		return false, fmt.Errorf("用户 %d 没有可用中心成员", userID)
	}

	pref.UnitOwningUserID = replacement.UnitOwningUserID
	if _, err := session.Table(new(usermodel.UserPref)).
		Where("user_id = ?", userID).
		Cols("unit_owning_user_id").
		Update(&pref); err != nil {
		return false, fmt.Errorf("修复用户 %d 的中心成员失败: %w", userID, err)
	}
	return true, nil
}

func repairUserDecks(
	session *xorm.Session,
	userID int,
	commonByUnitID map[int]unitmodel.CommonUnitData,
	unitByID map[int]usermodel.UserUnitData,
	unitByOwningID map[int]usermodel.UserUnitData,
) (int, error) {
	decks := []usermodel.UserDeck{}
	if err := session.Table(new(usermodel.UserDeck)).
		Where("user_id = ?", userID).
		OrderBy("deck_id ASC, id ASC").
		Find(&decks); err != nil {
		return 0, fmt.Errorf("读取用户 %d 的 user_deck 失败: %w", userID, err)
	}

	if len(decks) == 0 {
		deck := usermodel.UserDeck{
			DeckID:     1,
			DeckName:   "队伍A",
			MainFlag:   1,
			UserID:     userID,
			InsertDate: time.Now().Unix(),
		}
		if _, err := session.Insert(&deck); err != nil {
			return 0, fmt.Errorf("为用户 %d 创建默认卡组失败: %w", userID, err)
		}
		decks = append(decks, deck)
	}

	mainIndex := -1
	for i := range decks {
		if decks[i].MainFlag == 1 && mainIndex < 0 {
			mainIndex = i
		}
	}
	if mainIndex < 0 {
		mainIndex = 0
		decks[0].MainFlag = 1
		if _, err := session.Table(new(usermodel.UserDeck)).ID(decks[0].ID).Cols("main_flag").Update(&decks[0]); err != nil {
			return 0, fmt.Errorf("为用户 %d 设置主卡组失败: %w", userID, err)
		}
	}
	for i := range decks {
		if i != mainIndex && decks[i].MainFlag != 0 {
			decks[i].MainFlag = 0
			if _, err := session.Table(new(usermodel.UserDeck)).ID(decks[i].ID).Cols("main_flag").Update(&decks[i]); err != nil {
				return 0, fmt.Errorf("修复用户 %d 的卡组主标记失败: %w", userID, err)
			}
		}
	}

	repaired := 0
	for _, deck := range decks {
		changed, err := repairOneDeck(session, userID, deck, commonByUnitID, unitByID, unitByOwningID)
		if err != nil {
			return repaired, err
		}
		if changed {
			repaired++
		}
	}
	return repaired, nil
}

func repairOneDeck(
	session *xorm.Session,
	userID int,
	deck usermodel.UserDeck,
	commonByUnitID map[int]unitmodel.CommonUnitData,
	unitByID map[int]usermodel.UserUnitData,
	unitByOwningID map[int]usermodel.UserUnitData,
) (bool, error) {
	rows := []usermodel.UserDeckUnit{}
	if err := session.Table(new(usermodel.UserDeckUnit)).
		Where("user_deck_id = ?", deck.ID).
		OrderBy("position ASC, id ASC").
		Find(&rows); err != nil {
		return false, fmt.Errorf("读取用户 %d 卡组 %d 失败: %w", userID, deck.DeckID, err)
	}

	positions := map[int]struct{}{}
	resolved := make([]resolvedDeckUnit, 0, 9)
	valid := len(rows) == 9
	for _, row := range rows {
		if row.Position < 1 || row.Position > 9 {
			valid = false
			continue
		}
		if _, duplicate := positions[row.Position]; duplicate {
			valid = false
			continue
		}
		positions[row.Position] = struct{}{}

		userUnit, ok := unitByOwningID[row.UnitOwningUserID]
		if !ok || userUnit.UserID != userID {
			userUnit, ok = unitByID[row.UnitID]
		}
		if !ok {
			valid = false
			continue
		}
		common, ok := commonByUnitID[userUnit.UnitID]
		if !ok {
			valid = false
			continue
		}
		resolved = append(resolved, resolvedDeckUnit{UserUnit: userUnit, Common: common})
	}
	if len(positions) != 9 || len(resolved) != 9 {
		valid = false
	}

	if valid {
		changed := false
		for i, row := range rows {
			corrected := buildDeckUnit(deck.ID, userID, row.Position, resolved[i])
			corrected.ID = row.ID
			if !sameDeckUnit(row, corrected) {
				if _, err := session.Table(new(usermodel.UserDeckUnit)).ID(row.ID).AllCols().Update(&corrected); err != nil {
					return false, fmt.Errorf("修复用户 %d 卡组 %d 第 %d 位失败: %w", userID, deck.DeckID, row.Position, err)
				}
				changed = true
			}
		}
		return changed, nil
	}

	fallback, err := chooseFallbackDeckUnits(commonByUnitID, unitByID)
	if err != nil {
		return false, fmt.Errorf("用户 %d 卡组 %d 无法重建: %w", userID, deck.DeckID, err)
	}

	if _, err := session.Table(new(usermodel.UserDeckUnit)).Where("user_deck_id = ?", deck.ID).Delete(&usermodel.UserDeckUnit{}); err != nil {
		return false, fmt.Errorf("清理用户 %d 卡组 %d 旧成员失败: %w", userID, deck.DeckID, err)
	}
	for i, unit := range fallback {
		row := buildDeckUnit(deck.ID, userID, i+1, unit)
		if _, err := session.Insert(&row); err != nil {
			return false, fmt.Errorf("重建用户 %d 卡组 %d 第 %d 位失败: %w", userID, deck.DeckID, i+1, err)
		}
	}
	return true, nil
}

func chooseFallbackDeckUnits(
	commonByUnitID map[int]unitmodel.CommonUnitData,
	unitByID map[int]usermodel.UserUnitData,
) ([]resolvedDeckUnit, error) {
	result := make([]resolvedDeckUnit, 0, 9)
	for _, unitID := range canonicalDefaultDeckUnitIDs {
		userUnit, okUser := unitByID[unitID]
		common, okCommon := commonByUnitID[unitID]
		if okUser && okCommon {
			result = append(result, resolvedDeckUnit{UserUnit: userUnit, Common: common})
		}
	}
	if len(result) == 9 {
		return result, nil
	}

	result = result[:0]
	unitIDs := make([]int, 0, len(unitByID))
	for unitID := range unitByID {
		if _, ok := commonByUnitID[unitID]; ok {
			unitIDs = append(unitIDs, unitID)
		}
	}
	sort.Ints(unitIDs)
	for _, unitID := range unitIDs {
		result = append(result, resolvedDeckUnit{
			UserUnit: unitByID[unitID],
			Common:   commonByUnitID[unitID],
		})
		if len(result) == 9 {
			return result, nil
		}
	}
	return nil, fmt.Errorf("可用卡片不足 9 张")
}

func buildDeckUnit(deckRowID, userID, position int, unit resolvedDeckUnit) usermodel.UserDeckUnit {
	return usermodel.UserDeckUnit{
		UserDeckID:       deckRowID,
		UnitOwningUserID: unit.UserUnit.UnitOwningUserID,
		UnitID:           unit.UserUnit.UnitID,
		Position:         position,
		Level:            unit.Common.Level,
		LevelLimitID:     unit.Common.LevelLimitID,
		DisplayRank:      unit.UserUnit.DisplayRank,
		Love:             unit.Common.MaxLove,
		UnitSkillLevel:   unit.Common.UnitSkillLevel,
		IsRankMax:        unit.Common.IsRankMax,
		IsLoveMax:        unit.Common.IsLoveMax,
		IsLevelMax:       unit.Common.IsLevelMax,
		IsSigned:         unit.Common.IsSigned,
		BeforeLove:       unit.Common.MaxLove,
		MaxLove:          unit.Common.MaxLove,
		UserID:           userID,
		InsertDate:       time.Now().Unix(),
	}
}

func sameDeckUnit(a, b usermodel.UserDeckUnit) bool {
	return a.UserDeckID == b.UserDeckID &&
		a.UnitOwningUserID == b.UnitOwningUserID &&
		a.UnitID == b.UnitID &&
		a.Position == b.Position &&
		a.Level == b.Level &&
		a.LevelLimitID == b.LevelLimitID &&
		a.DisplayRank == b.DisplayRank &&
		a.Love == b.Love &&
		a.UnitSkillLevel == b.UnitSkillLevel &&
		a.IsRankMax == b.IsRankMax &&
		a.IsLoveMax == b.IsLoveMax &&
		a.IsLevelMax == b.IsLevelMax &&
		a.IsSigned == b.IsSigned &&
		a.BeforeLove == b.BeforeLove &&
		a.MaxLove == b.MaxLove &&
		a.UserID == b.UserID
}

func sanitizeDeckName(name string, deckID int) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	if deckID <= 1 {
		return "队伍A"
	}
	return fmt.Sprintf("队伍%d", deckID)
}
