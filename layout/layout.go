package layout

import (
	"errors"
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	go_wechat "github.com/oliverCJ/go-wechat"
	"github.com/oliverCJ/go-wechat/constants/types"
	"github.com/oliverCJ/go-wechat/services"
	"log"
	"strings"
	"time"
)

type Layout struct {
	userChatListBox      *widgets.List //聊天窗口
	userContactListBox   *widgets.List
	userGroupListBox *widgets.List
	headerBox            *widgets.Paragraph
	chatBox              *widgets.List //消息窗口
	inputBox             *widgets.Paragraph // 输入框

	contactList          services.ContactList // 通讯录
	chatUserList         []services.Member // 当前聊天用户

	chatMsgMapHistory				map[string][]string
	chatMsgMapNew				map[string][]string

	curChatFromName         string // 当前聊天者
	useInfo         services.User                        // 用户名字
	readMsg              <-chan services.Message         // 接收消息通道
	sendMsg              chan<- services.SendMessage     // 发送消息通道
	sendMsgResp          <-chan services.SendMessageResp // 发送消息响应通道
	closeChan            <-chan bool                     // 微信服务停止通知通道
	autoReply            bool
	globalUserMap        map[string]services.TinyMemberInfo
}

func NewLayout() *Layout {

	// 用户聊天
	userChatListBox := widgets.NewList()
	userChatListBox.TextStyle = ui.NewStyle(ui.ColorWhite)
	userChatListBox.WrapText = false
	userChatListBox.BorderStyle = ui.NewStyle(ui.ColorGreen)
	userChatListBox.SelectedRowStyle = ui.NewStyle(ui.ColorGreen)
	userChatListBox.SelectedRow = 0
	userChatListBox.Rows = []string{}

	// 联系人
	userContactListBox := widgets.NewList()
	userContactListBox.TextStyle = ui.NewStyle(ui.ColorWhite)
	userContactListBox.WrapText = false
	userContactListBox.BorderStyle = ui.NewStyle(ui.ColorGreen)
	userContactListBox.SelectedRowStyle = ui.NewStyle(ui.ColorGreen)
	userContactListBox.SelectedRow = 0
	userContactListBox.Rows = []string{}

	// 群组
	userGroupListBox := widgets.NewList()
	userGroupListBox.TextStyle = ui.NewStyle(ui.ColorWhite)
	userGroupListBox.WrapText = false
	userGroupListBox.BorderStyle = ui.NewStyle(ui.ColorGreen)
	userGroupListBox.SelectedRowStyle = ui.NewStyle(ui.ColorGreen)
	userGroupListBox.SelectedRow = 0
	userGroupListBox.Rows = []string{}


	// 聊天窗
	chatBox := widgets.NewList()
	chatBox.TextStyle = ui.NewStyle(ui.ColorWhite)
	chatBox.BorderStyle = ui.NewStyle(ui.ColorMagenta)
	chatBox.SelectedRowStyle = ui.NewStyle(ui.ColorGreen)
	chatBox.WrapText = false
	chatBox.SelectedRow = 0
	chatBox.Rows = []string{}

	inputBox := widgets.NewParagraph()
	inputBox.TextStyle = ui.NewStyle(ui.ColorWhite)
	inputBox.Title = "输入框"
	inputBox.BorderStyle = ui.NewStyle(ui.ColorCyan)

	headerBox := widgets.NewParagraph()
	headerBox.TextStyle = ui.NewStyle(ui.ColorWhite)
	headerBox.Border = false
	headerBox.BorderStyle = ui.NewStyle(ui.ColorWhite)
	headerBox.Text = "←/→ 向左或向右切换tab ↑/↓ 选择联系人 F1 选中聊天对象 CTRL+s/<Enter>发送消息 CTRL+c 退出 "

	return &Layout{
		userChatListBox:      userChatListBox,
		userContactListBox:   userContactListBox,
		userGroupListBox: userGroupListBox,
		chatBox:              chatBox,
		inputBox:             inputBox,
		headerBox:            headerBox,

		contactList: go_wechat.GetContact(),
		useInfo :go_wechat.GetUserInfo(),

		chatMsgMapHistory: make(map[string][]string),
		chatMsgMapNew: make(map[string][]string),

		readMsg: go_wechat.GetReadChan(),
		sendMsg: go_wechat.GetSendChan(),
		sendMsgResp: go_wechat.GetSendRespChan(),
		closeChan: go_wechat.GetCloseChan(),

		globalUserMap: go_wechat.GetGlobalMemberMap(),
	}
}

func (l *Layout) Init() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	userBoxList := [3]*widgets.List{}
	userBoxList[0] = l.userChatListBox
	userBoxList[1] = l.userContactListBox
	userBoxList[2] = l.userGroupListBox

	// 建立nickname map
	userNickNameMap := make(map[string]string)
	for _, v := range l.globalUserMap {
		runeStr := []rune(v.NickName)
		if len(runeStr) > 40 {
			userNickNameMap[string(runeStr[:40])] = v.UserName
		} else {
			userNickNameMap[v.NickName] = v.UserName
		}
	}

	// 聊天初始化
	chatUserIndexMap := l.initChatList()

	// 通讯录初始化
	for _, v := range l.contactList.MemberList {
		l.userContactListBox.Rows = append(l.userContactListBox.Rows, v.NickName)
	}

	// 群组初始化
	for _, v := range l.contactList.Group {
		l.userGroupListBox.Rows = append(l.userGroupListBox.Rows, v.NickName)
	}

	// 用户块tab
	tabpane := widgets.NewTabPane("聊天", "通讯录", "群组")
	tabpane.Border = true

	termWidth, termHeight := ui.TerminalDimensions()
	leftWidth := termWidth / 4

	gridL1 := ui.NewGrid()
	gridL1.SetRect(0, 0, leftWidth, termHeight)
	gridL1.Set(
		ui.NewRow(1.0,
			ui.NewRow(1.0,
				ui.NewCol(1.0,
					ui.NewRow(1.0/15, tabpane),
					ui.NewRow(14.0/15, l.userChatListBox),
				),
			),
		),
	)

	gridL2 := ui.NewGrid()
	gridL2.SetRect(0, 0, leftWidth, termHeight)
	gridL2.Set(
		ui.NewRow(1.0,
			ui.NewRow(1.0,
				ui.NewCol(1.0,
					ui.NewRow(1.0/15, tabpane),
					ui.NewRow(14.0/15, l.userContactListBox),
				),
			),
		),
	)

	gridL3 := ui.NewGrid()
	gridL3.SetRect(0, 0, leftWidth, termHeight)
	gridL3.Set(
		ui.NewRow(1.0,
			ui.NewRow(1.0,
				ui.NewCol(1.0,
					ui.NewRow(1.0/15, tabpane),
					ui.NewRow(14.0/15, l.userGroupListBox),
				),
			),
		),
	)

	gridRight := ui.NewGrid()
	gridRight.SetRect(leftWidth, 0, termWidth, termHeight)
	gridRight.Set(
		ui.NewRow(1.0,
			ui.NewCol(1.0,
				ui.NewRow(1.0/15, l.headerBox),
				ui.NewRow(10.0/15, l.chatBox),
				ui.NewRow(4.0/15, l.inputBox),
			),
		),
	)

	ui.Render(gridL1, gridRight)

	renderTab := func() {
		switch tabpane.ActiveTabIndex {
		case 0:
			for k, v := range l.userChatListBox.Rows {
				userFromName := userNickNameMap[strings.TrimPrefix(v, "*")]
				if userFromName != "" {
					if v2, ok2 := l.chatMsgMapNew[userFromName]; ok2 {
						if len(v2) > 0 {
							l.userChatListBox.Rows[k] = fmt.Sprintf("*%s", strings.TrimPrefix(v, "*"))
						} else {
							l.userChatListBox.Rows[k] = strings.TrimPrefix(v, "*")
						}
					}
				}
			}
			ui.Render(gridL1)
		case 1:
			ui.Render(gridL2)
		case 2:
			ui.Render(gridL3)
		}
	}

	go l.receiveMsg()

	uiEvents := ui.PollEvents()

	for {
		select {
		case <-time.Tick(20*time.Second):
			chatUserIndexMap = l.initChatList()
			ui.Clear()
			ui.Render(gridRight)
		case clo := <-l.closeChan:
			if clo {
				// 意外退出
				fmt.Print("意外退出")
				break
			}
		case e := <-uiEvents:
			switch e.ID {
			case "<C-c>":
				go_wechat.Stop()
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				leftWidth := payload.Width / 4
				gridL1.SetRect(0, 0, leftWidth, payload.Height)
				gridRight.SetRect(leftWidth, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(gridL1, gridRight)
				continue
			case "<Left>":
				tabpane.FocusLeft()
				ui.Clear()
				ui.Render(gridRight)
			case "<Right>":
				tabpane.FocusRight()
				ui.Clear()
				ui.Render(gridRight)
			case "<Down>":
				userBoxList[tabpane.ActiveTabIndex].ScrollDown()
			case "<Up>":
				userBoxList[tabpane.ActiveTabIndex].ScrollUp()
			case "<C-j>":
				l.chatBox.ScrollUp()
				ui.Render(gridRight)
			case "<C-k>":
				l.chatBox.ScrollDown()
				ui.Render(gridRight)
			case "<F1>":
				selectTitle := userBoxList[tabpane.ActiveTabIndex].Rows[userBoxList[tabpane.ActiveTabIndex].SelectedRow]
				trimTitle := strings.TrimPrefix(selectTitle, "*")
				// 通过昵称查找username
				userFromName := userNickNameMap[trimTitle]
				if _, ok := l.globalUserMap[userFromName]; ok {
					l.chatBox.Title = trimTitle
					l.curChatFromName = userFromName
					if l.globalUserMap[userFromName].Type == types.CONTACT_TYPE_GROUP {
						l.chatBox.Title += fmt.Sprintf("(%d)", l.globalUserMap[userFromName].MemberCount)
					}
					// 检查当前聊天对象是否已经在聊天组里
					if _, ok := chatUserIndexMap[trimTitle]; !ok {
						l.userChatListBox.Rows = append(l.userChatListBox.Rows, trimTitle)
						chatUserIndexMap[trimTitle] = len(l.userChatListBox.Rows) - 1
					}
					// 清空先
					l.chatBox.Rows = []string{}
					err := l.mergeMsg(l.curChatFromName)
					if err == nil {
						l.chatBox.Rows = l.chatMsgMapHistory[l.curChatFromName]
					}

					if len(l.chatBox.Rows) > 10 {
						l.chatBox.ScrollBottom()
					}
				}


				ui.Clear()
				ui.Render(gridRight)
			case "<C-s>", "<Enter>":
				content := strings.TrimSpace(l.inputBox.Text)
				if content != "" {
					l.sendMsg <- services.SendMessage{
						ToUserName: l.curChatFromName,
						Content: content,
						LocalID: fmt.Sprintf("%d", time.Now().Unix()),
					}
					resp := <- l.sendMsgResp
					if resp.BaseRequest.Ret == 0 {
						// TODO 发送成功
					}
					l.chatMsgMapHistory[l.curChatFromName] = append(l.chatMsgMapHistory[l.curChatFromName], fmt.Sprintf("我 说：%s", content))
					l.chatBox.Rows = l.chatMsgMapHistory[l.curChatFromName]
					l.inputBox.Text = ""
				}

				if len(l.chatBox.Rows) > 10 {
					l.chatBox.ScrollBottom()
				}

				ui.Clear()
				ui.Render(gridRight)
			case "<Backspace>":
				runes := []rune(l.inputBox.Text)
				if len(runes) > 0 {
					if len(runes) == 1 {
						l.inputBox.Text = ""
					} else {
						l.inputBox.Text = string(runes[:len(runes) - 1])
					}
					ui.Clear()
					ui.Render(gridRight)
				} else {
					continue
				}
			default:
				if e.Type == ui.KeyboardEvent {
					key := ""
					switch e.ID {
					case "<Enter>":
						key = ""
					case "<Space>":
						key = " "
					default:
						key = e.ID
					}
					l.addTextToPar(l.inputBox, key)
				}
			}
		}
		renderTab()
	}

}

func (l *Layout) receiveMsg() {
	for {
		select {
		case msg := <- l.readMsg :
			if msg.FormatContent != "" {
				msgSender := ""
				if msg.FromUserName == l.useInfo.UserName {
					// 自己在其他设备上发的消息
					msgSender = msg.ToUserName
					msg.FromUserNickName = "我"
				} else {
					msgSender = msg.FromUserName
				}
				if _, ok := l.chatMsgMapNew[msgSender]; !ok {
					temp := make([]string, 0)
					l.chatMsgMapNew[msgSender] = temp
				}
				temp2 := l.chatMsgMapNew[msgSender]
				if msg.FromUserName[:2] == "@@" {
					// 群组聊天找到说话人的信息
					if msg.RealUserNickName != "" {
						msg.FormatContent = fmt.Sprintf("%s 说： %s", msg.RealUserNickName, msg.FormatContent)
					}
					temp2 = append(temp2, msg.FormatContent)
				} else {
					if msg.FromUserNickName != "" {
						msg.FormatContent = fmt.Sprintf("%s 说： %s", msg.FromUserNickName, msg.FormatContent)
					}
					temp2 = append(temp2, msg.FormatContent)
				}
				l.chatMsgMapNew[msgSender] = temp2

				// 如果当前聊天对象，则合并新老消息
				if l.curChatFromName == msgSender {
					err := l.mergeMsg(msgSender)
					if err == nil {
						l.chatBox.Rows = l.chatMsgMapHistory[msgSender]
						if len(l.chatBox.Rows) > 10 {
							l.chatBox.ScrollBottom()
						}
						ui.Render(l.chatBox)
					}
				}
			}
		}
	}
}

// 合并新老消息
func (l *Layout) mergeMsg(fromName string) error {
	if _, ok := l.globalUserMap[fromName]; !ok {
		return errors.New("没有找到对应用户信息")
	}
	if _, ok := l.chatMsgMapHistory[fromName]; !ok {
		temp := make([]string, 0)
		l.chatMsgMapHistory[fromName] = temp
	}

	temp2 := l.chatMsgMapHistory[fromName]
	temp2 = append(temp2, l.chatMsgMapNew[fromName]...)

	l.chatMsgMapHistory[fromName] = temp2

	// 清空新消息
	l.chatMsgMapNew[fromName] = []string{}
	return nil
}

func (l *Layout) addTextToPar(p *widgets.Paragraph, text string) {
	p.Text += text
	ui.Render(p)
}

func (l *Layout) initChatList() map[string]int {
	l.userChatListBox.Rows = []string{}
	l.chatUserList = go_wechat.GetChatList()
	// 聊天初始化
	chatUserIndexMap := make(map[string]int)
	for _, v := range l.chatUserList {
		runeStr := []rune(v.NickName)
		if len(runeStr) > 40 {
			l.userChatListBox.Rows = append(l.userChatListBox.Rows, string(runeStr[:40]))
		} else {
			l.userChatListBox.Rows = append(l.userChatListBox.Rows, v.NickName)
		}
	}
	// username和索引建立MAP
	for k, v := range l.userChatListBox.Rows {
		chatUserIndexMap[v] = k
	}
	return chatUserIndexMap
}