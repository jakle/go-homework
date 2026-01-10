# 启动项目
go run main.go

# 测试完整流程
# 1. 注册用户
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com","password":"123456","role":"user"}'

# 2. 登录获取 token
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"123456"}'

# 3. 使用 token 访问受保护的接口
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/profile

# 4. 上传头像
curl --request POST \
  --url http://localhost:8080/api/v1/users/UploadAvatar \
  --header 'Accept: */*' \
  --header 'Accept-Encoding: gzip, deflate, br' \
  --header 'Authorization: Bearer <token>' \
  --header 'content-type: multipart/form-data' \
  --form 'avatar=@[object Object]'

# 5. 查询用户列表接口
curl --request GET \
  --url 'http://127.0.0.1:8080/api/v1/users?page=1&page_size=10' \
  --header 'Accept: application/x-protobuf' \
  --header 'Accept-Encoding: gzip, deflate, br' \
  --header 'Authorization: Bearer <token>' \
  --header 'Connection: keep-alive' \
  --header 'Content-Type: application/x-protobuf' \
  --header 'User-Agent: PostmanRuntime-ApipostRuntime/1.1.0'
