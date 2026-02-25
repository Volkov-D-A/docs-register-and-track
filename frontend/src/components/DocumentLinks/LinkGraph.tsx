import { useCallback, useEffect } from 'react';
import {
    ReactFlow,
    Controls,
    Background,
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

/**
 * Свойства компонента графа связей.
 */
interface LinkGraphProps {
    rootId: string;
    isLocked?: boolean;
}

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

    return {
        nodes: nodes.map(node => {
            const layer = layers[node.id] !== undefined ? layers[node.id] : 0;
            const nodesInThisLayer = nodesByLayer[layer];
            const index = nodesInThisLayer.indexOf(node);

            return {
                ...node,
                position: {
                    x: layer * 350,
                    y: index * 150 - (nodesInThisLayer.length * 150) / 2,
                },
                targetPosition: Position.Left,
                sourcePosition: Position.Right,
            };
        }),
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

            const initialNodes: Node[] = data.nodes.map((n: services.GraphNode) => ({
                id: n.id,
                data: {
                    label: (
                        <div style={{ padding: '10px', border: '1px solid #ddd', borderRadius: '5px', background: '#fff', fontSize: '12px', minWidth: '240px' }}>
                            <div style={{ fontWeight: 'bold', fontSize: '11px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }} title={`${n.type === 'incoming' ? 'Входящий' : 'Исходящий'} № ${n.label}`}>
                                {n.type === 'incoming' ? 'Входящий' : 'Исходящий'} № {n.label}
                            </div>
                            <div style={{ fontSize: '10px', color: '#666' }}>{n.date}</div>
                            <div style={{ fontSize: '10px', color: '#444', fontStyle: 'italic', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                {n.type === 'incoming' ? `От: ${n.sender}` : `Кому: ${n.recipient}`}
                            </div>
                            <div style={{ fontSize: '10px', marginTop: '5px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '220px' }} title={n.subject}>
                                {n.subject}
                            </div>
                        </div>
                    )
                },
                style: { width: 265, border: 'none', background: 'transparent' },
                position: { x: 0, y: 0 }, // will be calculated
            }));

            const initialEdges: Edge[] = data.edges.map((e: services.GraphEdge) => ({
                id: e.id,
                source: e.source,
                target: e.target,
                label: e.label === 'reply' ? 'Ответ' :
                    e.label === 'follow_up' ? 'Во исполнение' :
                        e.label === 'related' ? 'Связан' : e.label,
                type: 'smoothstep',
                animated: true,
                markerEnd: {
                    type: MarkerType.ArrowClosed,
                },
                style: { stroke: '#555' },
            }));

            const layouted = getLayoutedElements(initialNodes, initialEdges, rootId);
            setNodes(layouted.nodes);
            setEdges(layouted.edges);

            // Fit view after nodes are set and layouted
            setTimeout(() => fitView({ padding: 0.2, duration: 800 }), 100);

        }).catch(err => console.error("Failed to fetch graph:", err));
    }, [rootId, fitView]);

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
            fitView
            nodesDraggable={!isLocked}
            nodesConnectable={!isLocked}
            elementsSelectable={true}
        >
            <Controls />
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
