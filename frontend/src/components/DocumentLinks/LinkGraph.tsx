import { useCallback, useEffect } from 'react';
import {
    ReactFlow,
    Controls,
    Background,
    Panel,
    useNodesState,
    useEdgesState,
    addEdge,
    Connection,
    Edge,
    Node,
    Position,
    MarkerType,
    ReactFlowProvider,
    useReactFlow,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { GetDocumentFlow, services } from '../../types/link';
import { isIncomingKind } from '../../constants/documentKinds';
import { getDocumentLinkTypeLabel, getLinkedDocumentColor, getLinkedDocumentCounterpartyLabel, getLinkedDocumentLabel } from '../../config/documentLinkConfig';

/**
 * Свойства компонента графа связей.
 */
interface LinkGraphProps {
    rootId: string;
    isLocked?: boolean;
}

const NODE_COLORS = {
    incoming: {
        background: '#e6f4ff',
        border: '#91caff',
        accent: '#1677ff',
        badgeBackground: '#1677ff',
        meta: '#0958d9',
    },
    outgoing: {
        background: '#f6ffed',
        border: '#b7eb8f',
        accent: '#52c41a',
        badgeBackground: '#52c41a',
        meta: '#389e0d',
    },
} as const;

const getNodePalette = (type: string) => {
    return isIncomingKind(type) ? NODE_COLORS.incoming : NODE_COLORS.outgoing;
};

const NODE_WIDTH = 265;
const LAYER_HORIZONTAL_GAP = 460;
const LAYER_VERTICAL_GAP = 200;


const compareNodesForLayout = (a: Node, b: Node) => {
    const aLabel = typeof a.data?.documentNumber === 'string' ? a.data.documentNumber : '';
    const bLabel = typeof b.data?.documentNumber === 'string' ? b.data.documentNumber : '';
    const aDate = typeof a.data?.documentDate === 'string' ? a.data.documentDate : '';
    const bDate = typeof b.data?.documentDate === 'string' ? b.data.documentDate : '';
    const aKind = typeof a.data?.kindCode === 'string' ? a.data.kindCode : '';
    const bKind = typeof b.data?.kindCode === 'string' ? b.data.kindCode : '';

    if (aDate !== bDate) {
        return aDate.localeCompare(bDate);
    }
    if (aLabel !== bLabel) {
        return aLabel.localeCompare(bLabel);
    }
    if (aKind !== bKind) {
        return aKind.localeCompare(bKind);
    }

    return a.id.localeCompare(b.id);
};

const getLayoutedElements = (nodes: Node[], edges: Edge[], rootId: string) => {
    // Simple layered layout
    const layers: Record<string, number> = {};
    const visited = new Set<string>();
    const queue: { id: string, layer: number }[] = [{ id: rootId, layer: 0 }];

    visited.add(rootId);

    while (queue.length > 0) {
        const { id, layer } = queue.shift()!;
        layers[id] = layer;

        // Find connected nodes
        const connectedEdges = edges.filter(e => e.source === id || e.target === id);
        for (const edge of connectedEdges) {
            const nextId = edge.source === id ? edge.target : edge.source;
            if (!visited.has(nextId)) {
                visited.add(nextId);
                // If source is current, next is typically "next" in flow (positive layer)
                // If target is current, next is "previous" (negative layer)
                // But links can be arbitrary. Let's just do BFS distance for now.
                // Ideally we use direction.
                // let's assume flow is source -> target.
                if (edge.source === id) {
                    queue.push({ id: nextId, layer: layer + 1 });
                } else {
                    queue.push({ id: nextId, layer: layer - 1 });
                }
            }
        }
    }

    // Group by layer
    const nodesByLayer: Record<number, Node[]> = {};
    nodes.forEach(node => {
        const layer = layers[node.id] !== undefined ? layers[node.id] : 0;
        if (!nodesByLayer[layer]) nodesByLayer[layer] = [];
        nodesByLayer[layer].push(node);
    });

    Object.values(nodesByLayer).forEach((nodesInLayer) => {
        nodesInLayer.sort(compareNodesForLayout);
    });

    const layoutedNodes = nodes.map(node => {
            const layer = layers[node.id] !== undefined ? layers[node.id] : 0;
            const nodesInThisLayer = nodesByLayer[layer];
            const index = nodesInThisLayer.indexOf(node);

            return {
                ...node,
                position: {
                    x: layer * LAYER_HORIZONTAL_GAP,
                    y: index * LAYER_VERTICAL_GAP - (nodesInThisLayer.length * LAYER_VERTICAL_GAP) / 2,
                },
                targetPosition: Position.Left,
                sourcePosition: Position.Right,
            };
        });

    return {
        nodes: layoutedNodes,
        edges,
    };
};

/**
 * Компонент отрисовки содержимого графа связей (ReactFlow).
 */
const LinkGraphContent = ({ rootId, isLocked }: LinkGraphProps) => {
    const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
    const { fitView } = useReactFlow();

    useEffect(() => {
        if (!rootId) return;

        GetDocumentFlow(rootId).then((data: any) => {
            // Transform to ReactFlow format
            // Check if there are any nodes
            if (!data.nodes || data.nodes.length === 0) {
                setNodes([{
                    id: rootId,
                    data: { label: 'Current Document' },
                    position: { x: 0, y: 0 },
                    type: 'input'
                }]);
                setEdges([]);
                setTimeout(() => fitView({ padding: 0.2 }), 50);
                return;
            }

            const initialNodes: Node[] = data.nodes.map((n: services.GraphNode) => {
                const palette = getNodePalette(n.kindCode);
                const isRootNode = n.id === rootId;

                return {
                    id: n.id,
                    data: {
                        documentNumber: n.label,
                        documentDate: n.date,
                        kindCode: n.kindCode,
                        label: (
                            <div
                                style={{
                                padding: '12px',
                                    border: isRootNode ? `3px solid ${palette.accent}` : `1px solid ${palette.border}`,
                                    borderRadius: '10px',
                                    background: palette.background,
                                    boxShadow: isRootNode ? `0 0 0 3px ${palette.accent}22` : '0 6px 18px rgba(0, 0, 0, 0.06)',
                                    fontSize: '12px',
                                    minWidth: '240px',
                                }}
                            >
                                <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 6 }}>
                                    <span
                                        style={{
                                            display: 'inline-flex',
                                            alignItems: 'center',
                                            justifyContent: 'center',
                                            color: palette.accent,
                                            fontSize: '10px',
                                            fontWeight: 700,
                                            lineHeight: 1.4,
                                            textTransform: 'uppercase',
                                            letterSpacing: '0.04em',
                                        }}
                                    >
                                        {getLinkedDocumentLabel(n.kindCode)}
                                    </span>
                                </div>
                                <div
                                    style={{
                                        fontWeight: 'bold',
                                        fontSize: '11px',
                                        color: '#1f1f1f',
                                        whiteSpace: 'nowrap',
                                        overflow: 'hidden',
                                        textOverflow: 'ellipsis',
                                    }}
                                    title={`${getLinkedDocumentLabel(n.kindCode)} № ${n.label}`}
                                >
                                    № {n.label}
                                </div>
                                <div style={{ fontSize: '10px', color: palette.meta }}>{n.date}</div>
                                <div
                                    style={{
                                        fontSize: '10px',
                                        color: palette.meta,
                                        fontStyle: 'italic',
                                        whiteSpace: 'normal',
                                        overflowWrap: 'anywhere',
                                        wordBreak: 'break-word',
                                    }}
                                >
                                    {getLinkedDocumentCounterpartyLabel(n.kindCode, n.sender, n.recipient)}
                                </div>
                                <div
                                    style={{
                                        fontSize: '10px',
                                        marginTop: '5px',
                                        whiteSpace: 'normal',
                                        overflowWrap: 'anywhere',
                                        wordBreak: 'break-word',
                                        maxWidth: '220px',
                                    }}
                                    title={n.subject}
                                >
                                    {n.subject}
                                </div>
                            </div>
                        )
                    },
                    style: { width: NODE_WIDTH, border: 'none', background: 'transparent' },
                    position: { x: 0, y: 0 }, // will be calculated
                };
            });

            const initialEdges: Edge[] = data.edges.map((e: services.GraphEdge) => ({
                id: e.id,
                source: e.source,
                target: e.target,
                label: getDocumentLinkTypeLabel(e.label),
                type: 'smoothstep',
                animated: true,
                markerEnd: {
                    type: MarkerType.ArrowClosed,
                },
                style: { stroke: '#555' },
                labelStyle: {
                    fontSize: 12,
                    fontWeight: 600,
                    fill: '#434343',
                },
                labelBgStyle: {
                    fill: 'rgba(255, 255, 255, 0.96)',
                    stroke: '#d9d9d9',
                },
                labelBgPadding: [8, 4],
                labelBgBorderRadius: 6,
            }));

            const layouted = getLayoutedElements(initialNodes, initialEdges, rootId);
            setNodes(layouted.nodes);
            setEdges(layouted.edges);

            setTimeout(() => fitView({ padding: 0.2, duration: 0 }), 100);
        }).catch(err => console.error("Failed to fetch graph:", err));
    }, [rootId, fitView, setEdges, setNodes]);

    const onConnect = useCallback(
        (params: Connection) => setEdges((eds) => addEdge(params, eds)),
        [setEdges],
    );

    return (
        <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            nodesDraggable={!isLocked}
            nodesConnectable={!isLocked}
            elementsSelectable={true}
        >
            <Controls />
            <Panel position="top-left">
                <div
                    style={{
                        display: 'flex',
                        gap: 8,
                        padding: '8px 10px',
                        borderRadius: 10,
                        background: 'rgba(255, 255, 255, 0.92)',
                        border: '1px solid #f0f0f0',
                        boxShadow: '0 4px 14px rgba(0, 0, 0, 0.06)',
                    }}
                >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, color: '#595959' }}>
                        <span style={{ width: 10, height: 10, borderRadius: 999, background: getLinkedDocumentColor('incoming') }} />
                        {getLinkedDocumentLabel('incoming')}
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, color: '#595959' }}>
                        <span style={{ width: 10, height: 10, borderRadius: 999, background: getLinkedDocumentColor('outgoing') }} />
                        {getLinkedDocumentLabel('outgoing')}
                    </div>
                </div>
            </Panel>
            <Background gap={12} size={1} />
        </ReactFlow>
    );
};

/**
 * Обёртка для графа связей документа, предоставляющая контекст ReactFlow.
 */
export const LinkGraph = (props: LinkGraphProps) => {
    return (
        <div style={{ height: '600px', border: '1px solid #eee' }}>
            <ReactFlowProvider>
                <LinkGraphContent {...props} />
            </ReactFlowProvider>
        </div>
    );
};
