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
var token, guildID, announceChannelID string
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
}

func MemberBatch(ctx context.Context, m PubSubMessage) error {
	if err := dg.Open(); err != nil {
		log.Fatalln("Error opening connection,", err)
	} else {
		log.Println("success: Open connect")
	}

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
			// kick
			members, err := searchRoleMembers(mems, guild.ID, role.ID)
			if err != nil {
				return err
			}

			// userごとにkick
			for _, mem := range members {
				roleUserID := mem.User.ID
				userName := mem.User.Username
				byeMessage := "体験入部期間が終了したため"
				dg.GuildMemberDeleteWithReason(guildID, roleUserID, byeMessage)
				content := userName + " さんの体験入部期間が終了しました。"
				dg.ChannelMessageSend(announceChannelID, content)
			}
			// del role
			dg.GuildRoleDelete(guildID, role.ID)
			continue
		}

		// パースできて、かつ体験期間終了が2週間後の人は連絡
		if nowTime.AddDate(0, 0, 14).Format(layout) == trialTimeRole.Format(layout) {
			log.Println("kick after week: ", trialTimeRole)
			members, err := searchRoleMembers(mems, guild.ID, role.ID)
			if err != nil {
				return err
			}

			for _, mem := range members {
				mention := mem.Mention()
				content := mention + " さんの体験入部期間はあと2週間で終了します。\n今後も活動を続けたい場合は、ぜひ入部をお願いします。"
				dg.ChannelMessageSend(announceChannelID, content)
			}
			continue
		}

		// パースできて、かつ体験期間終了が1週間後の人は連絡
		if nowTime.AddDate(0, 0, 7).Format(layout) == trialTimeRole.Format(layout) {
			log.Println("kick after week: ", trialTimeRole)
			members, err := searchRoleMembers(mems, guild.ID, role.ID)
			if err != nil {
				return err
			}

			for _, mem := range members {
				mention := mem.Mention()
				content := mention + " さんの体験入部期間はあと1週間で終了します。\n今後も活動を続けたい場合は、ぜひ入部をお願いします。"
				dg.ChannelMessageSend(announceChannelID, content)
			}
			continue
		}

		// パースできて、かつ体験期間終了が明日の人がいる場合、確認用の連絡
		if nowTime.AddDate(0, 0, 1).Format(layout) == trialTimeRole.Format(layout) {
			log.Println("kick tommorow: ", trialTimeRole)
			members, err := searchRoleMembers(mems, guild.ID, role.ID)
			if err != nil {
				return err
			}

			for _, mem := range members {
				mention := mem.Mention()
				content := "自動通知: " + mention + " さんの体験入部期間が明日で終了します。\n部費の支払いが終わっている場合、" + mention + " さんの体験入部期間ロールを解除してください。"
				dg.ChannelMessageSend(announceChannelID, content)
			}
		}
		// パースできて、かつ体験入部期間が直近に迫っていない人はスキップ
	}

	dg.Close()

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
