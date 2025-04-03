package internal

import (
	_ "embed"

	"go.mau.fi/util/configupgrade"
	up "go.mau.fi/util/configupgrade"
	"maunium.net/go/mautrix/bridge/bridgeconfig"
)

//go:embed example-config.yaml
var ExampleConfig string

// var _ bridgekit.ConfigGetter = &Config{}
var _ bridgeconfig.BridgeConfig = &MyBridgeConfig{}

type MyBridgeConfig struct {
	SomeKey            string                           `yaml:"some_key"`
	Encryption         bridgeconfig.EncryptionConfig    `yaml:"encryption"`
	CommandPrefix      string                           `yaml:"command_prefix"`
	ManagementRoomText bridgeconfig.ManagementRoomTexts `yaml:"management_room_text"`
	DoublePuppetConfig bridgeconfig.DoublePuppetConfig  `yaml:",inline"`
	Permissions        bridgeconfig.PermissionConfig    `yaml:"permissions"`

	TestNested struct {
		SomeKey string `yaml:"some_key"`
	} `yaml:"bridge"`
}

func (m MyBridgeConfig) FormatUsername(username string) string {
	//TODO implement me
	return username
}

func (m MyBridgeConfig) GetEncryptionConfig() bridgeconfig.EncryptionConfig {
	//TODO implement me
	return m.Encryption
}

func (m MyBridgeConfig) GetCommandPrefix() string {
	//TODO implement me
	return m.CommandPrefix
}

func (m MyBridgeConfig) GetManagementRoomTexts() bridgeconfig.ManagementRoomTexts {
	//TODO implement me
	return m.ManagementRoomText
}

func (m MyBridgeConfig) GetDoublePuppetConfig() bridgeconfig.DoublePuppetConfig {
	//TODO implement me
	return m.DoublePuppetConfig
}

func (m MyBridgeConfig) GetResendBridgeInfo() bool {
	//TODO implement me
	return false
}

func (m MyBridgeConfig) EnableMessageStatusEvents() bool {
	//TODO implement me
	return false
}

func (m MyBridgeConfig) EnableMessageErrorNotices() bool {
	return true
}

func (m MyBridgeConfig) Validate() error {
	return nil
}

type Config struct {
	*bridgeconfig.BaseConfig `yaml:",inline"`

	SomeOtherSection struct {
		Key string `yaml:"key"`
	} `yaml:"some_other_section"`

	BridgeConfig *MyBridgeConfig `yaml:"bridge"`
}

func (m *Config) Bridge() bridgeconfig.BridgeConfig {
	return &MyBridgeConfig{}
}

func (m *Config) GetPtr(base *bridgeconfig.BaseConfig) any {
	m.BridgeConfig = &MyBridgeConfig{}
	m.BaseConfig = base
	m.BaseConfig.Bridge = m.BridgeConfig
	return m
}

func (m *Config) DoUpgrade(helper configupgrade.Helper) {
	bridgeconfig.Upgrader.DoUpgrade(helper)

	helper.Copy(up.Str, "bridge", "some_key")
	helper.Copy(up.Str, "some_other_section", "key")

	helper.Copy(up.Map, "bridge", "double_puppet_server_map")
	helper.Copy(up.Bool, "bridge", "double_puppet_allow_discovery")
	if legacySecret, ok := helper.Get(up.Str, "bridge", "login_shared_secret"); ok && len(legacySecret) > 0 {
		baseNode := helper.GetBaseNode("bridge", "login_shared_secret_map")
		baseNode.Map[helper.GetBase("homeserver", "domain")] = up.StringNode(legacySecret)
		baseNode.UpdateContent()
	} else {
		helper.Copy(up.Map, "bridge", "login_shared_secret_map")
	}
	if legacyPrivateChatPortalMeta, ok := helper.Get(up.Bool, "bridge", "private_chat_portal_meta"); ok {
		updatedPrivateChatPortalMeta := "default"
		if legacyPrivateChatPortalMeta == "true" {
			updatedPrivateChatPortalMeta = "always"
		}
		helper.Set(up.Str, updatedPrivateChatPortalMeta, "bridge", "private_chat_portal_meta")
	} else {
		helper.Copy(up.Str, "bridge", "private_chat_portal_meta")
	}
	helper.Copy(up.Str, "bridge", "management_room_text", "welcome")
	helper.Copy(up.Str, "bridge", "management_room_text", "welcome_connected")
	helper.Copy(up.Str, "bridge", "management_room_text", "welcome_unconnected")
	helper.Copy(up.Str|up.Null, "bridge", "management_room_text", "additional_help")

	helper.Copy(up.Bool, "bridge", "encryption", "allow")
	helper.Copy(up.Bool, "bridge", "encryption", "default")
	helper.Copy(up.Bool, "bridge", "encryption", "require")
	helper.Copy(up.Bool, "bridge", "encryption", "appservice")
	helper.Copy(up.Bool, "bridge", "encryption", "plaintext_mentions")
	helper.Copy(up.Bool, "bridge", "encryption", "delete_keys", "delete_outbound_on_ack")
	helper.Copy(up.Bool, "bridge", "encryption", "delete_keys", "dont_store_outbound")
	helper.Copy(up.Bool, "bridge", "encryption", "delete_keys", "ratchet_on_decrypt")
	helper.Copy(up.Bool, "bridge", "encryption", "delete_keys", "delete_fully_used_on_decrypt")
	helper.Copy(up.Bool, "bridge", "encryption", "delete_keys", "delete_prev_on_new_session")
	helper.Copy(up.Bool, "bridge", "encryption", "delete_keys", "delete_on_device_delete")
	helper.Copy(up.Bool, "bridge", "encryption", "delete_keys", "periodically_delete_expired")
	helper.Copy(up.Bool, "bridge", "encryption", "delete_keys", "delete_outdated_inbound")

	helper.Copy(up.Str, "bridge", "encryption", "verification_levels", "receive")
	helper.Copy(up.Str, "bridge", "encryption", "verification_levels", "send")
	helper.Copy(up.Str, "bridge", "encryption", "verification_levels", "share")

	legacyKeyShareAllow, ok := helper.Get(up.Bool, "bridge", "encryption", "key_sharing", "allow")
	if ok {
		helper.Set(up.Bool, legacyKeyShareAllow, "bridge", "encryption", "allow_key_sharing")
		legacyKeyShareRequireCS, legacyOK1 := helper.Get(up.Bool, "bridge", "encryption", "key_sharing", "require_cross_signing")
		legacyKeyShareRequireVerification, legacyOK2 := helper.Get(up.Bool, "bridge", "encryption", "key_sharing", "require_verification")
		if legacyOK1 && legacyOK2 && legacyKeyShareRequireVerification == "false" && legacyKeyShareRequireCS == "false" {
			helper.Set(up.Str, "unverified", "bridge", "encryption", "verification_levels", "share")
		}
	} else {
		helper.Copy(up.Bool, "bridge", "encryption", "allow_key_sharing")
	}

	helper.Copy(up.Bool, "bridge", "encryption", "rotation", "enable_custom")
	helper.Copy(up.Int, "bridge", "encryption", "rotation", "milliseconds")
	helper.Copy(up.Int, "bridge", "encryption", "rotation", "messages")
	helper.Copy(up.Bool, "bridge", "encryption", "rotation", "disable_device_change_key_rotation")

	helper.Copy(up.Map, "bridge", "permissions")
	helper.Copy(up.Bool, "bridge", "relay", "enabled")
	helper.Copy(up.Bool, "bridge", "relay", "admin_only")
	helper.Copy(up.Map, "bridge", "relay", "message_formats")
}
