package batch

import (
	"context"
	"log"
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

	dg, err = discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session, ", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuildMembers

	guildID = os.Getenv("GUILD_ID")
	announceChannelID = os.Getenv("ANNOUNCE_CHANNEL_ID")
	guestRoleID = os.Getenv("GUEST_ROLE_ID")
}

func MemberBatch(ctx context.Context, m PubSubMessage) error {
	err := dg.Open()
	if err != nil {
		panic(1)
	}
	defer dg.Close()

	// 日数ロールを探すため
	guild, err := dg.Guild(guildID)
	if err != nil {
		return err
	}

	// member のロール取得のため
	mems, err := dg.GuildMembers(guildID, "", 1000)
	if err != nil {
		return err
	}

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	layout := "2006/1/2"

	nowTime := time.Now().In(jst)
	for _, role := range guild.Roles {
		trialTimeRole, err := time.ParseInLocation(layout, role.Name, jst)

		// パース出来ないロールは関係ないのでスキップ
		if err != nil {
			continue
		}

		// パースできて、かつ体験期間が今日より前の人はkickして、ロールを消す
		if trialTimeRole.Before(nowTime) {
			log.Println("kick: ", trialTimeRole.Format(layout))
			members, err := searchRoleMembers(mems, guild.ID, role.ID)
			if err != nil {
				return err
			}

			// userごとにkick
			for _, mem := range members {
				mention := mem.Mention()
				// Guestロールの削除
				err = dg.GuildMemberRoleRemove(guildID, mem.User.ID, guestRoleID)
				if err != nil {

					log.Println(err)
				}
				// 体験入部期間終了のお知らせ
				content := mention + " さんの体験入部期間が終了しました。"
				dg.ChannelMessageSend(announceChannelID, content)
			}
			// del role
			dg.GuildRoleDelete(guildID, role.ID)
			continue
		}

		// パースできて、かつ体験期間終了が明日の人がいる場合、確認用の連絡
		if trialTimeRole.Format(layout) == nowTime.AddDate(0, 0, 1).Format(layout) {
			log.Println("role remove after tommorow: ", trialTimeRole)
			members, err := searchRoleMembers(mems, guild.ID, role.ID)
			if err != nil {
				return err
			}

			for _, mem := range members {
				mention := mem.Mention()
				content := mention + "さんの体験入部期間は明日で終了します。"
				dg.ChannelMessageSend(announceChannelID, content)
			}
		}
		// パースできて、かつ体験入部期間が直近に迫っていない人はスキップ
	}

	log.Println("Batch Success!")
	return nil
}

func searchRoleMembers(mems []*discordgo.Member, guildID, roleID string) (members []*discordgo.Member, err error) {
	// 引数で受け取ったロールIDを持つ メンバー(部員) をmembersスライスにappend
	// 体験入部期間のリミット権限は最大で1人1つのため、そのメンバーにリミット権限が既にあればそれ以上調べる必要はない
	for _, mem := range mems {
		for _, memRoleID := range mem.Roles {
			if memRoleID == roleID {
				members = append(members, mem)
				break
			}
		}
	}
	return members, nil
}
