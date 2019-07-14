# go-slack-block-kit

[slack block kit](https://api.slack.com/block-kit)のサンプルコードです  
SlackからGitHub branch の一覧を取得・選択してJenkinsのジョブを実行します  

# 使い方

AWS Secrets Manager を利用して各種Tokenを設定しています  
その他の方法で設定する場合は `config.go` を書き換えてください

`config.go` で設定している各種値の説明  

```
OauthToken          string `json:"SLACK_BOT_OAUTH_TOKEN"`             // Slack App で発行したBot用OAuthToken
VerificationToken   string `json:"SLACK_BOT_API_VERIFICATION_TOKEN"`  // Slack App で発行した検証用Token
JenkinsBotUserToken string `json:"JENKINS_BOT_USER_TOKEN"`            // Jenkinsのジョブ実行ユーザーToken
JenkinsJobToken     string `json:"JENKINS_JOB_TOKEN"`                 // Jenkinsのジョブをリモート実行するためのToken
GitHubBotUserToken  string `json:"GITHUB_BOT_USER_TOKEN"`             // デプロイ対象のリポジトリのRead権限を持つGitHub Personal Access Token
```

環境変数の設定  
`docker-compose.yml` の編集か `export xxx` で設定してください  
```
AWS_ACCESS_KEY_ID:                  // AWS Access Key
AWS_SECRET_ACCESS_KEY:              // AWS Secret Key
AWS_SECRET_NAME: hoge/slack         // AWS Secrets Manager name
BOT_ID: XXX                         // Slack App で発行したBotユーザーのID
GITHUB_OWNER: autumo                // デプロイ対象のリポジトリのGitHub Owner
JENKINS_HOST: jenkins.example.com   // jenkins host
```

実行方法  
```
docker-compose build && docker-compose up
```

Block Kit のレスポンスを受けるためのローカルサーバー構築は[こちら](https://api.slack.com/tutorials/tunneling-with-ngrok)
