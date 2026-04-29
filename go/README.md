# OMDb API 澶?Key 浠ｇ悊绠＄悊鍣紙Go锛?

杩欐槸涓€涓叕缃戝彲閮ㄧ讲鐨?OMDb API 浠ｇ悊鏈嶅姟锛氬鎴风璇锋眰鏍煎紡淇濇寔 OMDb 瀹樻柟椋庢牸锛屼絾 `apikey` 浣跨敤浣犺嚜宸卞彂鏀剧殑鏈嶅姟 key銆傛湇鍔＄浼氫粠 `omdb_keys.txt` 涓疆璇㈤€夋嫨 OMDb 瀹樻柟 key锛岃姹傚畼鏂规帴鍙ｅ悗鍘熸牱杩斿洖銆?

## 宸叉敮鎸佺殑 API

- `GET /`锛歄MDb 鏁版嵁 API锛屽吋瀹瑰畼鏂?query 鍙傛暟銆?
- `GET /api`锛氭暟鎹?API 鍒悕銆?
- `GET /poster`锛歄MDb poster API 浠ｇ悊銆?
- `GET /docs`锛氶潤鎬佽姹傛枃妗ｉ〉闈€?
- `GET /health`锛氬仴搴锋鏌ャ€?
- `GET /admin/stats`锛氭煡鐪?key 姹犵姸鎬侊紝闇€瑕?admin key銆?
- `POST /admin/reload`锛氶噸杞?`omdb_keys.txt` 鍜?`client_keys.txt`锛岄渶瑕?admin key銆?

鏁版嵁 API 閫忎紶鎵€鏈夊畼鏂瑰弬鏁帮紝渚嬪锛歚i`銆乣t`銆乣s`銆乣type`銆乣y`銆乣plot`銆乣r`銆乣callback`銆乣v`銆乣page`銆乣Season`銆乣Episode` 绛夈€?

## Key 鏂囦欢

### `omdb_keys.txt`

OMDb 瀹樻柟 key 姹狅紝涓€琛屼竴涓細

```txt
omdb_key_1
omdb_key_2
omdb_key_3
```

### `client_keys.txt`

浣犲彂缁欒皟鐢ㄦ柟鐨勬湇鍔¤闂?key锛屼竴琛屼竴涓細

```txt
client_key_1
client_key_2
```

娌℃湁瀹㈡埛绔?key 鎴?key 閿欒鏃讹紝浠ｇ悊 API 浼氱洿鎺ヨ繑鍥?`401`锛屼笉浼氳姹?OMDb銆?

## 閰嶇疆

澶嶅埗绀轰緥锛?

```powershell
Copy-Item .env.example .env
Copy-Item omdb_keys.example.txt omdb_keys.txt
Copy-Item client_keys.example.txt client_keys.txt
```

缂栬緫 `.env`锛?

```env
LISTEN_ADDR=:8080
OMDB_KEYS_FILE=omdb_keys.txt
CLIENT_KEYS_FILE=client_keys.txt
ADMIN_KEY=change_me_admin_key
HTTP_TIMEOUT=10s
KEY_COOLDOWN=5m
CORS_ORIGIN=*
```

## 鍚姩

```powershell
go run .
```

鏋勫缓锛?

```powershell
go build -o omdb-api-manager.exe .
```

璁块棶鏂囨。锛?

```text
http://localhost:8080/docs
```

## 璇锋眰绀轰緥

### 鎸夋爣棰樻煡璇?

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Inception&plot=full
```

### 鎸?IMDb ID 鏌ヨ

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&i=tt1375666
```

### 鎼滅储

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&s=Batman&page=2
```

### 鍓ч泦瀛?闆?

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Game%20of%20Thrones&Season=1
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Game%20of%20Thrones&Season=1&Episode=1
```

### XML / JSONP

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Inception&r=xml
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Inception&callback=myCallback
```

### Poster API

```text
GET http://localhost:8080/poster?apikey=YOUR_CLIENT_KEY&i=tt1375666
```

### Header 浼?key

涔熷彲浠ユ妸鏈嶅姟 key 鏀惧湪璇锋眰澶撮噷锛?

```text
GET /?t=Inception HTTP/1.1
X-API-Key: YOUR_CLIENT_KEY
```

鎴栵細

```text
Authorization: Bearer YOUR_CLIENT_KEY
```

## 绠＄悊鎺ュ彛

### 鏌ョ湅鐘舵€?

```text
GET http://localhost:8080/admin/stats?admin_key=ADMIN_KEY
```

### 閲嶈浇 key 鏂囦欢

```text
POST http://localhost:8080/admin/reload?admin_key=ADMIN_KEY
```

## Docker 閮ㄧ讲

```powershell
docker build -t omdb-api-manager .
docker run -d --name omdb-api-manager `
  -p 8080:8080 `
  -v ${PWD}/omdb_keys.txt:/app/omdb_keys.txt:ro `
  -v ${PWD}/client_keys.txt:/app/client_keys.txt:ro `
  --env-file .env `
  omdb-api-manager
```

鍏綉寤鸿鏀惧湪 Caddy/Nginx 鍚庨潰鍋?HTTPS銆?

## 璐熻浇涓庡閿?

- OMDb key 榛樿杞浣跨敤銆?
- 濡傛灉鏌愪釜 OMDb key 杩斿洖棰濆害鑰楀敖銆佹棤鏁?key銆?29銆?xx 鎴栬秴鏃讹紝浼氳繘鍏ュ喎鍗达紝鑷姩灏濊瘯涓嬩竴涓?key銆?
- 鏅€氫笟鍔￠敊璇笉浼氬垏 key锛屼緥濡?`Movie not found!` 浼氬師鏍疯繑鍥炵粰瀹㈡埛绔€?
- 瀹㈡埛绔?key 涓嶉檺娴併€?

