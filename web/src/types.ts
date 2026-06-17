export interface ToolCallInfo {
  id: string;
  name: string;
  input: any;
}

export interface Analysis {
  tool_calls: ToolCallInfo[];
  model: string;
  input_tokens: number;
  output_tokens: number;
  cache_creation_input_tokens?: number;
  cache_read_input_tokens?: number;
  stop_reason: string;
  duration_ms: number;
  mcp_servers?: string[];
  skills_used?: string[];
}

export interface RequestEvent {
  id: string;
  timestamp: string;
  request: {
    model: string;
    system?: any;
    messages: any[];
    tools?: any[];
    max_tokens?: number;
    stream?: boolean;
    raw_json: any;
  };
  response?: {
    id: string;
    type: string;
    role: string;
    content: any[];
    model: string;
    stop_reason: string;
    usage: { input_tokens: number; output_tokens: number };
    raw_json: any;
  };
  analysis?: Analysis;
}

export interface WSEvent {
  type: 'request.new' | 'request.complete' | 'tool.call';
  data: RequestEvent;
}

export interface ToolNode {
  name: string;
  count: number;
  lastUsed?: Date;
  category: 'tool' | 'mcp' | 'skill';
}
