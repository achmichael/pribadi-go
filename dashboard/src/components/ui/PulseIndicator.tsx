import React from 'react';

type PulseStatus = 'connected' | 'disconnected' | 'connecting';

interface PulseIndicatorProps {
  status: PulseStatus;
  className?: string;
}

const PulseIndicator: React.FC<PulseIndicatorProps> = ({ status, className = '' }) => {
  const getDotColor = () => {
    switch (status) {
      case 'connected':
        return 'bg-accent-primary';
      case 'connecting':
        return 'bg-accent-warm';
      case 'disconnected':
      default:
        return 'bg-ink-muted';
    }
  };

  const getRingColor = () => {
    switch (status) {
      case 'connected':
        return 'bg-accent-primary';
      case 'connecting':
        return 'bg-accent-warm';
      case 'disconnected':
      default:
        return 'transparent';
    }
  };

  const getPulseAnimation = () => {
    switch (status) {
      case 'connected':
        return 'pulse-ring-slow';
      case 'connecting':
        return 'pulse-ring-fast';
      case 'disconnected':
      default:
        return '';
    }
  };

  return (
    <div className={`relative flex items-center justify-center w-3 h-3 ${className}`}>
      {/* Outer pulsing ring */}
      {status !== 'disconnected' && (
        <div 
          className={`absolute inset-0 rounded-full ${getRingColor()} ${getPulseAnimation()}`}
        />
      )}
      
      {/* Inner solid dot */}
      <div 
        className={`relative w-2 h-2 rounded-full ${getDotColor()}`}
      />
    </div>
  );
};

export default PulseIndicator;
