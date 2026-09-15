import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_table.module.css";

export interface Admin2Column<T> {
  key: string;
  label: string;
  render: (row: T) => ReactNode;
  className?: string;
}

interface Admin2TableProps<T> {
  columns: Array<Admin2Column<T>>;
  rows: T[];
  rowKey: (row: T) => string;
  emptyRow?: ReactNode;
  className?: string;
}

export const Admin2Table = <T,>({
  columns,
  rows,
  rowKey,
  emptyRow,
  className
}: Admin2TableProps<T>) => (
  <div className={joinClassNames(styles.wrap, className)}>
    <table className={styles.table}>
      <thead>
        <tr>
          {columns.map((column) => (
            <th key={column.key} className={column.className}>
              {column.label}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.length === 0 && emptyRow ? (
          <tr>
            <td className={styles.emptyCell} colSpan={columns.length}>
              {emptyRow}
            </td>
          </tr>
        ) : (
          rows.map((row) => (
            <tr key={rowKey(row)}>
              {columns.map((column) => (
                <td key={column.key} className={column.className}>
                  {column.render(row)}
                </td>
              ))}
            </tr>
          ))
        )}
      </tbody>
    </table>
  </div>
);