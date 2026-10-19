import { useState } from 'react';
import DeadlineControls from '../features/deadlines/DeadlineControls';
import type { DeadlineView } from '../features/deadlines/deadlineEvents';
import DocumentViewModal from '../components/DocumentViewModal';
export default function DeadlineControlsPage({ initialView }: { initialView?: DeadlineView }) {
  const [document, setDocument] = useState<{ id: string; kind: string }>();
  return <div className="deadline-controls-page">
    <DeadlineControls initialView={initialView} onOpenDocument={(id, kind) => setDocument({ id, kind })} />
    {document && <DocumentViewModal open documentId={document.id} documentKind={document.kind} onCancel={() => setDocument(undefined)} />}
  </div>;
}
