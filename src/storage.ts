import * as vscode from "vscode";
import * as fs from "fs/promises";
import * as path from "path";
import { PlannerData } from "./types";

const DATA_DIR = ".eisenhower-todo";
const DATA_FILE = "data.json";

export class LocalJsonStorage {
  private readonly filePath: string;

  constructor(workspaceRoot: string) {
    this.filePath = path.join(workspaceRoot, DATA_DIR, DATA_FILE);
  }

  async load(): Promise<PlannerData> {
    try {
      const raw = await fs.readFile(this.filePath, "utf8");
      return JSON.parse(raw) as PlannerData;
    } catch (error: any) {
      if (error?.code !== "ENOENT") throw error;
      const initial: PlannerData = {
        version: 1,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
        actionPoints: [],
        archive: []
      };
      await this.save(initial);
      return initial;
    }
  }

  async save(data: PlannerData): Promise<void> {
    await fs.mkdir(path.dirname(this.filePath), { recursive: true });
    const tempPath = `${this.filePath}.tmp`;
    await fs.writeFile(tempPath, JSON.stringify(data, null, 2), "utf8");
    await fs.rename(tempPath, this.filePath);
  }

  getPath(): string {
    return this.filePath;
  }

  static resolveWorkspaceRoot(): string {
    const folder = vscode.workspace.workspaceFolders?.[0];
    if (!folder) {
      throw new Error("Open a workspace folder before using Eisenhower Todo.");
    }
    return folder.uri.fsPath;
  }
}
