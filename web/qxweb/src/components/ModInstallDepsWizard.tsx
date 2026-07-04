import { useCallback, useEffect, useRef, useState } from 'react';
import {
  type ModProjectType,
  type ModSource,
  type ModVersion,
} from '@/api/client';
import { ModInstallDepsModal, type DepsModalConfirmResult, type InstallItem } from '@/components/ModInstallDepsModal';
import { useInstanceMods } from '@/components/InstanceModsContext';
import {
  buildNestedDepsWizardSteps,
  dedupeInstallItemsByProject,
  type ModDepsWizardStep,
} from '@/lib/modCatalogDeps';

type Props = {
  open: boolean;
  source: ModSource;
  projectId: string;
  projectName: string;
  version: ModVersion;
  resourceType: ModProjectType;
  installedProjectIds: Set<string>;
  confirming: boolean;
  onCancel: () => void;
  onComplete: (items: InstallItem[]) => void;
};

function stepKey(step: ModDepsWizardStep): string {
  return `${step.source}:${step.projectId}`;
}

function splitStepItems(items: InstallItem[], step: ModDepsWizardStep) {
  const primary = items.find(
    (item) => item.source === step.source && item.projectId === step.projectId,
  );
  const dependencies = items.filter(
    (item) => !(item.source === step.source && item.projectId === step.projectId),
  );
  return { primary, dependencies };
}

function stepToInstallItem(step: ModDepsWizardStep): InstallItem {
  return {
    source: step.source,
    projectId: step.projectId,
    projectName: step.projectName,
    version: step.version,
    resourceType: step.resourceType,
  };
}

export function ModInstallDepsWizard({
  open,
  source,
  projectId,
  projectName,
  version,
  resourceType,
  installedProjectIds,
  confirming,
  onCancel,
  onComplete,
}: Props) {
  const { instance } = useInstanceMods();
  const rootRef = useRef<ModDepsWizardStep | null>(null);
  const [modalQueue, setModalQueue] = useState<ModDepsWizardStep[]>([]);
  const [currentStep, setCurrentStep] = useState<ModDepsWizardStep | null>(null);
  const [accumulatedItems, setAccumulatedItems] = useState<InstallItem[]>([]);
  const [deferredPrimaries, setDeferredPrimaries] = useState<InstallItem[]>([]);

  const resetWizard = useCallback(() => {
    rootRef.current = null;
    setModalQueue([]);
    setCurrentStep(null);
    setAccumulatedItems([]);
    setDeferredPrimaries([]);
  }, []);

  useEffect(() => {
    if (!open) {
      resetWizard();
      return;
    }
    const root: ModDepsWizardStep = {
      source,
      projectId,
      projectName,
      version,
      resourceType,
    };
    rootRef.current = root;
    setModalQueue([root]);
    setCurrentStep(root);
    setAccumulatedItems([]);
    setDeferredPrimaries([]);
  }, [open, projectId, projectName, resetWizard, resourceType, source, version]);

  const handleStepConfirm = async ({ items, selectedRequired }: DepsModalConfirmResult) => {
    if (!currentStep) return;

    const { primary, dependencies } = splitStepItems(items, currentStep);
    const remainingQueue = modalQueue.slice(1);
    const nextAccumulated = dedupeInstallItemsByProject([...accumulatedItems, ...dependencies]);

    const nestedSteps = await buildNestedDepsWizardSteps(selectedRequired, installedProjectIds, {
      loader: instance.loader,
      mcVersion: instance.mc_version,
    });

    if (nestedSteps.length > 0) {
      if (primary) {
        setDeferredPrimaries((prev) => dedupeInstallItemsByProject([...prev, primary]));
      }
      const nextQueue = [...nestedSteps, ...remainingQueue];
      setAccumulatedItems(nextAccumulated);
      setModalQueue(nextQueue);
      setCurrentStep(nextQueue[0] ?? null);
      return;
    }

    const withPrimary = primary
      ? dedupeInstallItemsByProject([...nextAccumulated, primary])
      : nextAccumulated;

    if (remainingQueue.length > 0) {
      setAccumulatedItems(withPrimary);
      setModalQueue(remainingQueue);
      setCurrentStep(remainingQueue[0] ?? null);
      return;
    }

    onComplete(dedupeInstallItemsByProject([...withPrimary, ...deferredPrimaries]));
  };

  if (!open || !currentStep) {
    return null;
  }

  const isRootStep =
    rootRef.current != null && stepKey(currentStep) === stepKey(rootRef.current);
  const hasMoreSteps = modalQueue.length > 1;

  return (
    <ModInstallDepsModal
      open={open}
      source={currentStep.source}
      projectId={currentStep.projectId}
      projectName={currentStep.projectName}
      version={currentStep.version}
      resourceType={currentStep.resourceType}
      installedProjectIds={installedProjectIds}
      confirming={confirming}
      nested={!isRootStep}
      hasMoreSteps={hasMoreSteps}
      onCancel={onCancel}
      onConfirm={(result) => void handleStepConfirm(result)}
    />
  );
}

export { stepToInstallItem };
