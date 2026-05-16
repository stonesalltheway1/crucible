-- build_recall_envelope.lua — atomic per-tenant recall envelope build
--
-- KEYS[1] = crucible:{tenant_id}:task:{task_id}:plan
-- KEYS[2] = crucible:{tenant_id}:task:{task_id}:branch
-- KEYS[3] = crucible:{tenant_id}:task:{task_id}:tools_list
-- KEYS[4] = crucible:{tenant_id}:recall:{cache_key}
--
-- ARGV[1] = max_tool_calls
--
-- Returns a JSON-encoded envelope { plan, branch, tools[], cache_hit }.

local plan = redis.call('GET', KEYS[1]) or ''
local branch = redis.call('GET', KEYS[2]) or ''
local n = tonumber(ARGV[1]) or 50
local tools = redis.call('LRANGE', KEYS[3], 0, n - 1)
local cached = redis.call('GET', KEYS[4])

local out = {
    plan = plan,
    branch = branch,
    tools = tools,
    cache_hit = cached ~= false,
    cached = cached or ''
}
return cjson.encode(out)
