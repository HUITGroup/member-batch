package batch

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

type PubSubMessage struct {
	Data []byte `json:"data"`
}

var dg *discordgo.Session
var token, guildID, announceChannelID, guestRoleID string
var err error

func init() {
	token = os.Getenv("DISCORD_TOKEN")
	guildID = os.Getenv("GUILD_ID")
	announceChannelID = os.Getenv("ANNOUNCE_CHANNEL_ID")
	guestRoleID = os.Getenv("GUEST_ROLE_ID")

	dg, err = discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}
}

func MemberBatch(ctx context.Context, m PubSubMessage) error {
	err := dg.Open()
	if err != nil {
		return err
	}
	defer dg.Close()

	guild, err := dg.Guild(guildID)
	if err != nil {
		return err
	}
	guildMembers, err := dg.GuildMembers(guildID, "", 1000)
	if err != nil {
		return err
	}

	layout := "2006/1/2"
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)

	year, month, day := time.Now().Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, jst)

	for _, guildRole := range guild.Roles {
		trialLimit, err := time.ParseInLocation(layout, guildRole.Name, jst)
		if err != nil {
			// パース出来ないロールは関係ないのでスキップ
			continue
		}

		if trialLimit.Equal(today.AddDate(0, 0, 7)) {
			roleMembers := findRoleMembers(guildMembers, guild.ID, guildRole.ID)
			if err != nil {
				return err
			}
			err = notifyMembersKickDay(roleMembers, 7)
			if err != nil {
				return err
			}
			continue
		}

		// 体験入部期間を過ぎていた場合
		if trialLimit.Before(today) {
			roleMembers := findRoleMembers(guildMembers, guild.ID, guildRole.ID)
			if err != nil {
				return err
			}
			err = removeRolesFromMembers(roleMembers, guild.ID, guildRole.ID, guestRoleID)
			if err != nil {
				return err
			}
			err = dg.GuildRoleDelete(guildID, guildRole.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func notifyMembersKickDay(members []*discordgo.Member, day int) error {
	for _, mem := range members {
		mention := mem.Mention()
		content := fmt.Sprintf(mention+"さんの体験入部期間はあと %d 日で終了します。本入部希望の場合は #本入部について の手順に沿って部費をお納めください。", day)
		_, err := dg.ChannelMessageSend(announceChannelID, content)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeRolesFromMembers(members []*discordgo.Member, guildID, guildRoleID, guestRoleID string) error {
	for _, mem := range members {
		content := mem.Mention() + " さんの体験入部期間が終了しました。"
		_, err = dg.ChannelMessageSend(announceChannelID, content)
		if err != nil {
			return err
		}

		err = dg.GuildMemberRoleRemove(guildID, mem.User.ID, guestRoleID)
		if err != nil {
			return err
		}

		err = dg.GuildMemberRoleRemove(guildID, mem.User.ID, guildRoleID)
		if err != nil {
			return err
		}
	}
	return nil
}

func findRoleMembers(mems []*discordgo.Member, guildID, roleID string) (members []*discordgo.Member) {
	// 体験入部期間のリミット権限は最大で1人1つのため、そのメンバーにリミット権限が既にあればそれ以上調べる必要はない
	for _, mem := range mems {
		for _, memRoleID := range mem.Roles {
			if memRoleID == roleID {
				members = append(members, mem)
				break
			}
		}
	}
	return members
}
