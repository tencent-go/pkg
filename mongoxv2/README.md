
## 功能
- 連接管理
- 泛型 序列化、反序列化
- 空文檔行為 getOne返回自定義錯誤 `ErrNotFound` list返回空數組
- create自動設置createdAt updatedAt version id
- update自動設置updatedAt version
- update忽略零值可選
- update樂觀鎖機制
- watch的封裝 可選持久化ResumeToken 同名consumer透過鎖獨佔