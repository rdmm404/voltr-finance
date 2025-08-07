- [x] add user id automatically into context for LLM
  - [x] include relevant information like household id
  - [ ] and other household members
- [ ] reply in discord in between agent func calls
- [ ] if message comes from server -> household by default, if comes from pm -> personal by default if not otherwise specified
- [] store transactions first then ask details
  - return created transaction ids from db
  - add update transactions tool
  - add instructions to system prompt
- [] improve system prompt
- [] add discord server id into households table to link server to household
- [] overall improve context management
- [] make users only be in a single household
- [] Add more context from the incoming message, like message time

- [] tools to get transactions using natural language
- [] implementpaid/owed functionality
  - [] add user default owed amount into household_user
- [] improve discord interactions


MAYBE
- [ ] Handle simultaneous messages 