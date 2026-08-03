package startup

func StartUp() {
	CreateTables()
	MigrateLegacyUnitTables()
	LoadUnitData()
	ReconcileLegacyOwnershipReferences()
	ReconcileLegacyUserState()
	CreateDefaultUser()
	EnsureDefaultFriends()
}
