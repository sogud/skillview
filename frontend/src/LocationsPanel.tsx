import { useEffect, useState, type CSSProperties } from "react";
import { Archive, Copy, Folder, Link, Trash2 } from "lucide-react";
import * as api from "./api";
import type { Skill, Occurrence, LocationReport } from "./api";
import { IconButton, Modal } from "./components";
import { useLocale } from "./locales";

export type LocationSelection = {
  copies: Skill[];
  issues?: { sourceId: string; path: string }[];
};

// Short labels are display-only; actions always use the resolver's full paths.
function LocationPath({ path, prefix }: { path: string; prefix: string }) {
  if (!prefix || !path.startsWith(prefix)) return <code dir="ltr">{path}</code>;
  return (
    <details className="location-path">
      <summary title={path}><code dir="ltr">…/{path.slice(prefix.length)}</code></summary>
      <code className="location-full-path" dir="ltr">{path}</code>
    </details>
  );
}

export function LocationsPanel({
  selection,
  onClose,
  onArchive,
  onTrash,
  onRemoveLink,
  onFolders,
}: {
  selection: LocationSelection;
  onClose: () => void;
  onArchive: (skill: Skill) => void;
  onTrash: (skill: Skill) => void;
  onRemoveLink: (skill: Skill, entry: Occurrence) => void;
  onFolders: () => void;
}) {
  const { t, error: message } = useLocale();
  const [reports, setReports] = useState<LocationReport[]>([]);
  const [failure, setFailure] = useState("");
  const [notice, setNotice] = useState("");
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    let active = true;
    const requests = selection.issues
      ? selection.issues.map((issue) =>
          api.Locations("", issue.sourceId, issue.path),
        )
      : selection.copies.map((copy) => api.Locations(copy.id, "", ""));
    void Promise.all(requests)
      .then((values) => {
        if (active) setReports(values);
      })
      .catch((error) => {
        if (active) setFailure(message(error));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [selection, message]);
  async function act(
    copyIndex: number,
    chainIndex: number,
    nodeIndex: number,
    copyPath: boolean,
  ) {
    setFailure("");
    try {
      await api.LocationAction(
        selection.copies[copyIndex]?.id ?? "",
        selection.issues?.[copyIndex]?.sourceId ?? "",
        selection.issues?.[copyIndex]?.path ?? "",
        chainIndex,
        nodeIndex,
        copyPath,
      );
      if (copyPath) setNotice(t("Copied"));
    } catch (error) {
      setFailure(message(error));
    }
  }
  const paths = reports.flatMap((report) =>
    (report.chains ?? []).flatMap((chain) =>
      (chain.nodes ?? []).map((node) => node.path),
    ),
  ).filter((path) => path.startsWith("/"));
  // Preserve at least the containing folder and basename in every short label.
  let common = paths[0]?.split("/").slice(0, -2) ?? [];
  for (const path of paths.slice(1)) {
    const parts = path.split("/").slice(0, -2);
    const different = common.findIndex((part, index) => part !== parts[index]);
    if (different !== -1) common = common.slice(0, different);
  }
  const prefix = common.length > 2 ? `${common.join("/")}/` : "";
  const statuses: Record<string, string> = {
    missing: "Missing link target",
    cycle: "Link cycle",
    denied: "Access denied",
    changed: "Location changed; scan again.",
    limit: "Link limit reached",
    unavailable: "Unavailable",
  };
  return (
    <Modal
      className="locations-dialog"
      title={t("Locations and references")}
      onClose={onClose}
    >
      <div className="modal-body">
        <p className="quiet-note">
          {t(
            "Counts include discovered locations only. Copies are shown separately.",
          )}
        </p>
        <p className="location-legend"><span aria-hidden="true" />{t("Link")}</p>
        {loading && <p role="status">{t("Loading…")}</p>}
        {reports.map((report, copyIndex) => {
          const skill = selection.copies[copyIndex];
          return (
            <section
              className="location-copy"
              key={skill?.id ?? selection.issues?.[copyIndex]?.path}
            >
              {skill && (
                <header className="location-copy-head">
                  <strong dir="auto">{skill.name}</strong>
                  <span>{t(skill.editable ? "Writable" : "Read-only")}</span>
                </header>
              )}
              {(report.chains ?? []).map((chain, chainIndex) => {
                const nodes = chain.nodes ?? [];
                const entry = (skill?.occurrences ?? []).find(
                  (entry) => entry.path === chain.path && entry.linked,
                );
                let depth = 0;
                return (
                  <div className="location-chain" key={chain.path}>
                    <ol className="location-tree">
                      {nodes.map((node, nodeIndex) => {
                        // Only a link node establishes an edge to its next target.
                        // Other resolver steps are not directory parent/child edges.
                        const linkedFromPrevious = nodeIndex > 0 && nodes[nodeIndex - 1].kind === "link";
                        const previousDepth = depth;
                        depth = linkedFromPrevious ? Math.min(depth + 1, 5) : 0;
                        const linksToNext = node.kind === "link" && nodeIndex < nodes.length - 1;
                        const kind = node.kind === "link" ? "Link"
                          : node.kind === "entry" ? "Entry"
                          : node.kind === "unavailable" ? "Unavailable"
                          : "Original location";
                        return (
                          <li className={`${linkedFromPrevious ? "linked-target" : ""} ${linksToNext ? "has-link-target" : ""}`} key={nodeIndex}
                            style={{ marginInlineStart: `${depth * 16}px`, "--tree-step": `${(depth - previousDepth) * 16}px` } as CSSProperties}>
                            <div className="location-tree-row">
                              {node.kind === "link" ? <Link size={14} aria-hidden="true" /> : <Folder size={14} aria-hidden="true" />}
                              <LocationPath path={node.path} prefix={prefix} />
                              <span className="location-kind">{t(kind)}</span>
                              <span className="location-node-actions">
                                <IconButton icon={Copy} label={t("Copy")} onClick={() => void act(copyIndex, chainIndex, nodeIndex, true)} />
                                <IconButton icon={Folder} label={t("Show in Finder")} onClick={() => void act(copyIndex, chainIndex, nodeIndex, false)} />
                                {nodeIndex === 0 && skill && entry ? (
                                  <button className="icon-button" disabled={!chain.removable}
                                    aria-label={t("Remove link")}
                                    title={t(chain.removable ? "Remove link" : "select the link's containing folder with editing allowed")}
                                    onClick={() => onRemoveLink(skill, entry)}>
                                    <Trash2 size={14} />
                                  </button>
                                ) : <span className="location-action-space" aria-hidden="true" />}
                              </span>
                            </div>
                          </li>
                        );
                      })}
                    </ol>
                    {chain.status !== "ready" && (
                      <p className="form-error" role="status">{t(statuses[chain.status] || "Unavailable")}</p>
                    )}
                  </div>
                );
              })}
              {skill && (
                <div className="location-management">
                  {(!skill.editable ||
                    (skill.occurrences ?? []).some((entry) => entry.linked)) && (
                    <p className="quiet-note">
                      {t(
                        skill.editable
                          ? "These links will point to a missing Skill:"
                          : "Archive and delete require an editable original folder. Linked entries remain read-only.",
                      )}
                    </p>
                  )}
                  {skill.editable &&
                    (skill.occurrences ?? [])
                      .filter((entry) => entry.linked)
                      .map((entry) => (
                        <code
                          className="location-impact"
                          dir="ltr"
                          key={entry.path}
                        >
                          {entry.path}
                        </code>
                      ))}
                  <div>
                    <button onClick={() => onArchive(skill)}>
                      <Archive size={14} />
                      {t("Archive Skill")}
                    </button>
                    <button onClick={() => onTrash(skill)}>
                      <Trash2 size={14} />
                      {t("Move Skill to Trash")}
                    </button>
                    {!skill.editable && (
                      <button onClick={onFolders}>
                        {t("Choose editable folder")}
                      </button>
                    )}
                  </div>
                </div>
              )}
            </section>
          );
        })}
        {failure && (
          <p className="form-error" role="alert">
            {failure}
          </p>
        )}
        {notice && <p role="status">{notice}</p>}
        <div className="modal-actions">
          <button onClick={onClose}>{t("Close")}</button>
        </div>
      </div>
    </Modal>
  );
}
