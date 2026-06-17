import type { RequestEvent } from '../types';
import './RequestList.css';

interface RequestListProps {
  events: RequestEvent[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

export function RequestList({ events, selectedId, onSelect }: RequestListProps) {
  return (
    <div className="request-list">
      <h3 className="request-list-title">Requests</h3>
      <div className="request-list-items">
        {[...events].reverse().map((event) => (
          <div
            key={event.id}
            className={`request-item ${selectedId === event.id ? 'selected' : ''}`}
            onClick={() => onSelect(event.id)}
          >
            <div className="request-item-header">
              <span className="request-item-model">{event.request.model}</span>
              <span className="request-item-time">
                {new Date(event.timestamp).toLocaleTimeString()}
              </span>
            </div>
            <div className="request-item-summary">
              {event.analysis ? (
                <>
                  <span>{(event.analysis.input_tokens || 0) + (event.analysis.output_tokens || 0)} tokens</span>
                  {(event.analysis.cache_read_input_tokens || 0) > 0 && (
                    <span className="request-item-cache">
                      ⚡ {Math.round(((event.analysis.cache_read_input_tokens || 0) / ((event.analysis.input_tokens || 0) + (event.analysis.cache_read_input_tokens || 0) + (event.analysis.cache_creation_input_tokens || 0))) * 100)}%
                    </span>
                  )}
                  {(event.analysis.tool_calls?.length || 0) > 0 && (
                    <span className="request-item-tools">
                      {event.analysis.tool_calls.length} tool{event.analysis.tool_calls.length > 1 ? 's' : ''}
                    </span>
                  )}
                  <span className={`request-item-status ${event.analysis.stop_reason || ''}`}>
                    {event.analysis.stop_reason || ''}
                  </span>
                </>
              ) : (
                <span className="request-item-pending">streaming...</span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
