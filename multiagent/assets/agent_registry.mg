% Agent Roles
agent_role("planner").
agent_role("executor").
agent_role("reviewer").
agent_role("researcher").
agent_role("coordinator").
agent_role("supervisor").
agent_role("orchestrator").

% Role Capabilities
role_capability("planner", "task_decomposition").
role_capability("planner", "goal_analysis").
role_capability("planner", "plan_generation").
role_capability("planner", "llm_reasoning").

role_capability("executor", "action_execution").
role_capability("executor", "llm_generation").
role_capability("executor", "tool_invocation").

role_capability("reviewer", "validation").
role_capability("reviewer", "quality_check").
role_capability("reviewer", "llm_reasoning").

role_capability("researcher", "knowledge_retrieval").
role_capability("researcher", "vector_search").

% Role Inheritance
role_inherits("supervisor", "planner").
role_inherits("supervisor", "executor").
role_inherits("supervisor", "reviewer").

% Task Requirements
task_requires_capability("generate_document", "llm_generation").
task_requires_capability("validate_output", "validation").
task_requires_capability("create_plan", "plan_generation").

% Agent Registry
agent("planner-001", "planner").
agent("executor-001", "executor").
agent("executor-002", "executor").
agent("reviewer-001", "reviewer").
agent("researcher-001", "researcher").

% Agent Capabilities
agent_capability("planner-001", "llm_reasoning").
agent_capability("planner-001", "task_decomposition").
agent_capability("executor-001", "llm_generation").
agent_capability("executor-001", "tool_invocation").
agent_capability("executor-002", "llm_generation").
agent_capability("reviewer-001", "llm_reasoning").
agent_capability("reviewer-001", "validation").
agent_capability("researcher-001", "knowledge_retrieval").

% Agent Config
agent_config("planner-001", "model", "gpt-4o").
agent_config("executor-001", "model", "gpt-4o").
agent_config("reviewer-001", "model", "gpt-4o").

% Agent Status
agent_status("planner-001", "available").
agent_status("executor-001", "available").
agent_status("executor-002", "available").
agent_status("reviewer-001", "available").
agent_status("researcher-001", "available").

% Workflows
workflow("content-pipeline", "Content Generation", "v1.0").
workflow("simple-gen", "Simple Generation", "v1.0").

% Workflow Nodes
workflow_node("content-pipeline", "research", "agent", "researcher").
workflow_node("content-pipeline", "plan", "agent", "planner").
workflow_node("content-pipeline", "execute", "agent", "executor").
workflow_node("content-pipeline", "review", "agent", "reviewer").

workflow_node("simple-gen", "generate", "agent", "executor").
workflow_node("simple-gen", "review", "agent", "reviewer").

% Workflow Edges
workflow_edge("content-pipeline", "research", "plan").
workflow_edge("content-pipeline", "plan", "execute").
workflow_edge("content-pipeline", "execute", "review").

workflow_edge("simple-gen", "generate", "review").
