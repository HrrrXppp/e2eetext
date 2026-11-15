package e2ee

const (
	MaxMessageDataLen     = 153_600
	MaxIdentityKeyJSONLen = 8_192
	MaxWrapJSONLen        = 16_384
	// MaxAddMembersBatch caps how many users one add-members request may add.
	MaxAddMembersBatch = 100
)
