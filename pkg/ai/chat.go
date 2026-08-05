package ai

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

func Chat(ctx context.Context, client *Client, in io.Reader, out io.Writer) {
	fmt.Fprintf(out, "\n%s\n\n", "🤖 Gitee CLI AI : 你好，我是 Gitee CLI AI 小助手，欢迎向我提问（输入 exit 退出）")

	var history []Message
	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprintf(out, "👤 User : ")
		if !scanner.Scan() {
			fmt.Fprintf(out, "\n👋 Bye\n")
			return
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "exit" {
			fmt.Fprintf(out, "\n🤖 Gitee CLI AI : 👋 Bye, See you next time！\n\n")
			return
		}

		history = append(history, Message{Role: "user", Content: input})

		fmt.Fprintf(out, "\n🤖 Gitee CLI AI : ")
		reply, err := client.CompleteStream(ctx, history, func(chunk string) {
			fmt.Fprint(out, chunk)
		})
		fmt.Fprintf(out, "\n\n")

		if err != nil {
			fmt.Fprintf(out, "❌ 请求失败: %v\n\n", err)
			history = history[:len(history)-1]
			continue
		}

		history = append(history, Message{Role: "assistant", Content: reply})
	}
}
