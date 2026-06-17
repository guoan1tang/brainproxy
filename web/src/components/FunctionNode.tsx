import { motion } from 'framer-motion';
import './FunctionNode.css';

interface FunctionNodeProps {
  name: string;
  icon?: string;
  count: number;
  isActive: boolean;
  color?: string;
  onClick?: () => void;
}

export function FunctionNode({ name, icon, count, isActive, color = '#6366f1', onClick }: FunctionNodeProps) {
  return (
    <motion.div
      className={`function-node ${isActive ? 'active' : ''} ${onClick ? 'clickable' : ''}`}
      initial={{ opacity: 0.4, scale: 0.9 }}
      animate={{
        opacity: isActive ? 1 : 0.5,
        scale: isActive ? 1.1 : 0.95,
        boxShadow: isActive
          ? `0 0 20px ${color}80, 0 0 40px ${color}40`
          : '0 0 0px transparent',
      }}
      whileHover={onClick ? { scale: 1.2 } : undefined}
      whileTap={onClick ? { scale: 0.95 } : undefined}
      transition={{ duration: 0.3 }}
      style={{ '--node-color': color } as React.CSSProperties}
      onClick={onClick}
    >
      <div className="node-icon">{icon || '🔧'}</div>
      {name && <div className="node-name">{name}</div>}
      {count > 0 && <div className="node-count">{count}</div>}
    </motion.div>
  );
}
