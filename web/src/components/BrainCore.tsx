import { motion } from 'framer-motion';
import './BrainCore.css';

interface BrainCoreProps {
  totalRequests: number;
  totalTokens: number;
  isActive: boolean;
}

export function BrainCore({ totalRequests, totalTokens, isActive }: BrainCoreProps) {
  return (
    <div className="brain-core">
      <motion.div
        className="brain-glow"
        animate={{
          scale: isActive ? [1, 1.2, 1] : 1,
          opacity: isActive ? [0.3, 0.8, 0.3] : 0.2,
        }}
        transition={{
          duration: 1.5,
          repeat: isActive ? Infinity : 0,
          ease: 'easeInOut',
        }}
      />
      <motion.div
        className="brain-icon"
        animate={{
          scale: isActive ? [1, 1.05, 1] : 1,
        }}
        transition={{
          duration: 2,
          repeat: Infinity,
          ease: 'easeInOut',
        }}
      >
        🧠
      </motion.div>
      <div className="brain-stats">
        <div className="brain-stat">
          <span className="brain-stat-value">{totalRequests}</span>
          <span className="brain-stat-label">requests</span>
        </div>
        <div className="brain-stat">
          <span className="brain-stat-value">{totalTokens > 1000 ? `${(totalTokens / 1000).toFixed(1)}k` : totalTokens}</span>
          <span className="brain-stat-label">tokens</span>
        </div>
      </div>
    </div>
  );
}
