import { useState } from 'react';
import { WelcomeScreen } from './Onboarding/components/WelcomeScreen';
import { ConnectPlatform } from './Onboarding/components/ConnectPlatform';
import { PersonaSetup } from './Onboarding/components/PersonaSetup';
import { SuccessScreen } from './Onboarding/components/SuccessScreen';

export default function Onboarding() {
  const [step, setStep] = useState(1);
  const totalSteps = 4;

  const handleNext = () => {
    if (step < totalSteps) setStep(step + 1);
  };

  const handleBack = () => {
    if (step > 1) setStep(step - 1);
  };

  return (
    <div className="min-h-screen flex flex-col bg-canvas">
      {/* Progress Header */}
      <div className="bg-surface border-b border-ink-primary/10 sticky top-0 z-10 shadow-sm">
        <div className="max-w-3xl mx-auto px-4 py-4 flex flex-col items-center justify-center">
          <div className="w-full flex items-center justify-between gap-2 max-w-sm">
            {Array.from({ length: totalSteps }).map((_, i) => (
              <div key={i} className="flex-1 h-2 bg-brand-100 rounded-full overflow-hidden">
                <div 
                  className={`h-full bg-accent-primary rounded-full transition-all duration-500 ease-out ${
                    i < step ? 'w-full' : 'w-0'
                  }`}
                />
              </div>
            ))}
          </div>
          <span className="text-xs font-semibold text-ink-muted/70 mt-3 uppercase tracking-widest">
            Langkah {step} dari {totalSteps}
          </span>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col items-center justify-center p-4">
        <div className="w-full max-w-2xl bg-surface rounded-3xl shadow-sm border border-ink-primary/10 p-8 sm:p-12 overflow-hidden relative">
          
          {step === 1 && <WelcomeScreen onNext={handleNext} />}
          {step === 2 && <ConnectPlatform onNext={handleNext} onBack={handleBack} />}
          {step === 3 && <PersonaSetup onNext={handleNext} onBack={handleBack} />}
          {step === 4 && <SuccessScreen />}

        </div>
      </div>
    </div>
  );
}
