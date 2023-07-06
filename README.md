# [WIP] Simple ActivityPub Server
まともに書き直すのは完成してから

参考サイト:
* https://qiita.com/wakin/items/94a0ff3f32f842b18a25  
Userとして認識されるまで
* https://qiita.com/wakin/items/28cacf78095d853bfa67  
Noteの返答,FollowのAccept,Followerへの転送まで
* https://zenn.dev/myuon/articles/8b49a0c8afdc52  
上記2つの補足など
* https://scrapbox.io/lacolaco/ActivityPub_%E5%AE%9F%E8%A3%85%E3%83%A1%E3%83%A2  
流れでざっと書いてあるやつ
* https://qiita.com/nullkal/items/accc5d62836a930b3cd9  
ActivityPubの定義(和訳版?)
* https://webfinger.net/lookup/  
Webfingerの確認したサイト
* https://qiita.com/keitaj/items/00aede60e64e8eebbb8a
golangでの鍵署名
* https://asnokaze.hatenablog.com/entry/2020/01/07/012014
Signature(署名)の方法
* https://dinochiesa.github.io/httpsig/
署名のテスター

## Webfinger
Webfingerは最初に`https://${Domain}/.well-known/host-meta`に問い合わせる  
問い合わせた後、`host-meta`の`template=${URL}`にアクセスする(`${uri}`に`acct:${User}@${Domain}`)

## Person
ユーザーに関する詳細な情報(通知やフォローなど)を書く
```json
{
  "@context": [ // 不明
      "https://www.w3.org/ns/activitystreams",
      "https://w3id.org/security/v1"
  ],

  "url": "https://${Domain}/${User}", // ユーザーのプロフィールリンク?
  "type": "Person", // activitystreamsのPerson 明記

  "followers": "https://${Domain}/${User}/followers", // フォロワー 一覧
  "following": "https://${Domain}/${User}/following", // フォロー 一覧

  "id": "https://${Domain}/${User}", // ユーザーのID?
  "preferredUsername": "${User}", // ユーザーID
  "name": "${User}",  // 表示名
  "icon": { // Icon
      "mediaType": "image/png", // Iconの mime Type
      "type": "Image",
      "url": "https://${Domain}/${User}/icon" // IconのURL
  },
  "summary": "@${User} using on ${Domain}", // 概要

  "inbox": "https://${Domain}/${User}/inbox", // このユーザーへの宛先
  "outbox": "https://${Domain}/${User}/outbox" // このユーザーの発信元
}
```


openssl genrsa -out privatekey.pem 2048  
秘密鍵  
openssl rsa -in privatekey.pem -outform pem -pubout -out publickey.pem  
公開鍵  