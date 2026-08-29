import { TreeView as ArkTreeView, createTreeCollection } from '@ark-ui/solid/tree-view'
import { For, Show, createMemo, type JSX } from 'solid-js'
import { treeItemClass } from './ark/styles'
import { cn } from './cn'
import type { TreeNode as KgTreeNode } from './types'

type ArkNode = {
  id: string
  name: string
  meta?: string
  children?: ArkNode[]
}

type Props<T = unknown> = {
  nodes: KgTreeNode<T>[]
  selectedId?: string | null
  onSelect?: (node: KgTreeNode<T>) => void
}

function convertNodes<T>(nodes: KgTreeNode<T>[]): ArkNode[] {
  return nodes.map((node) => ({
    id: node.id,
    name: node.label,
    meta: node.meta,
    children: node.children?.length ? convertNodes(node.children) : undefined,
  }))
}

function TreeNode(props: { node: ArkNode; indexPath: number[]; meta?: boolean }) {
  return (
    <ArkTreeView.NodeProvider node={props.node} indexPath={props.indexPath}>
      <ArkTreeView.NodeContext>
        {(nodeState) => (
          <Show
            when={nodeState().branch}
            fallback={
              <ArkTreeView.Item class={treeItemClass}>
                <ArkTreeView.ItemText>{props.node.name}</ArkTreeView.ItemText>
                <Show when={props.node.meta}>
                  <span class="bg-[var(--kg-surface-muted)] px-2 py-0.5 text-xs text-[var(--kg-text-muted)]">
                    {props.node.meta}
                  </span>
                </Show>
              </ArkTreeView.Item>
            }
          >
            <ArkTreeView.Branch>
              <ArkTreeView.BranchControl class={treeItemClass}>
                <ArkTreeView.BranchTrigger class="flex w-full items-center gap-2">
                  <ArkTreeView.BranchIndicator class="text-xs">▾</ArkTreeView.BranchIndicator>
                  <ArkTreeView.BranchText>{props.node.name}</ArkTreeView.BranchText>
                  <Show when={props.node.meta}>
                    <span class="bg-[var(--kg-surface-muted)] px-2 py-0.5 text-xs text-[var(--kg-text-muted)]">
                      {props.node.meta}
                    </span>
                  </Show>
                </ArkTreeView.BranchTrigger>
              </ArkTreeView.BranchControl>
              <ArkTreeView.BranchContent>
                <For each={props.node.children ?? []}>
                  {(child, index) => <TreeNode node={child} indexPath={[...props.indexPath, index()]} />}
                </For>
              </ArkTreeView.BranchContent>
            </ArkTreeView.Branch>
          </Show>
        )}
      </ArkTreeView.NodeContext>
    </ArkTreeView.NodeProvider>
  )
}

export function TreeView<T = unknown>(props: Props<T>): JSX.Element {
  const rootNode = createMemo<ArkNode>(() => ({
    id: '__root__',
    name: '',
    children: convertNodes(props.nodes),
  }))

  const collection = createMemo(() =>
    createTreeCollection({
      rootNode: rootNode(),
      nodeToValue: (node) => node.id,
      nodeToString: (node) => node.name,
      nodeToChildren: (node) => node.children ?? [],
    }),
  )

  const selectedValue = () => (props.selectedId ? [props.selectedId] : [])

  return (
    <div class="overflow-hidden border border-[var(--kg-border)] bg-[var(--kg-surface)]">
      <Show
        when={props.nodes.length > 0}
        fallback={<p class="px-12 py-12 text-center text-sm text-[var(--kg-text-muted)]">No items</p>}
      >
        <ArkTreeView.Root
          collection={collection()}
          selectedValue={selectedValue()}
          selectionMode="single"
          onSelectionChange={(details) => {
            const id = details.selectedValue[0]
            if (!id || id === '__root__') return
            const find = (nodes: KgTreeNode<T>[]): KgTreeNode<T> | undefined => {
              for (const node of nodes) {
                if (node.id === id) return node
                const child = node.children?.length ? find(node.children) : undefined
                if (child) return child
              }
              return undefined
            }
            const hit = find(props.nodes)
            if (hit) props.onSelect?.(hit)
          }}
        >
          <ArkTreeView.Tree>
            <For each={rootNode().children ?? []}>
              {(node, index) => <TreeNode node={node} indexPath={[index()]} />}
            </For>
          </ArkTreeView.Tree>
        </ArkTreeView.Root>
      </Show>
    </div>
  )
}
