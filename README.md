# pikaGPT
![image](https://user-images.githubusercontent.com/17543997/228567135-28adf1f7-fb75-4b00-9025-fe8f6a77af89.png)
Discord上で動かせる、Go言語で書かれたBotです。手元で動かす場合は、以下のように.envファイルを作り、
中にDiscordのBOTトークン(BOTTOKEN=)と、openapiのAPIキー(APIKEY=)を適切に埋めてください。

![image](https://user-images.githubusercontent.com/17543997/228567438-b0185853-cfe3-4479-b75a-afba10fba272.png)

機能は以下の通りです。

`/gpt <対話内容>` gpt-3.5-turboと対話します。過去に対話した内容も覚えます(文脈を理解できる)。

`/gpt reset` 履歴をリセットします。

`/gpt debug` 履歴と最後の一回で消費した金額、最後にリセットされてから消費された金額を確認できます。

- 1回に送信できるトークン最大値の上限を超えた場合、古い履歴を削除してリトライします。
