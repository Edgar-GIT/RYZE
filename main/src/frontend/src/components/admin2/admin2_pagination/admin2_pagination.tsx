import { ChevronLeft, ChevronRight } from "lucide-react";

import { Button } from "@/components/button/button";

import styles from "./admin2_pagination.module.css";

interface Admin2PaginationProps {
  page: number;
  totalPages: number;
  pageSize: number;
  totalItems: number;
  onChange: (page: number) => void;
}

export const Admin2Pagination = ({ page, totalPages, pageSize, totalItems, onChange }: Admin2PaginationProps) => {
  const from = Math.min(totalItems, (page - 1) * pageSize + 1);
  const to = Math.min(totalItems, page * pageSize);

  return (
    <div className={styles.footer}>
      <p className={styles.info}>
        Showing {totalItems === 0 ? "0" : `${from}–${to}`} of {totalItems}
      </p>
      <div className={styles.controls}>
        <Button
          variant="secondary"
          size="small"
          icon={<ChevronLeft size={14} />}
          iconPosition="left"
          disabled={page <= 1}
          onClick={() => onChange(page - 1)}
        >
          Previous
        </Button>
        <span className={styles.pageIndicator}>
          Page {page} of {Math.max(totalPages, 1)}
        </span>
        <Button
          variant="secondary"
          size="small"
          icon={<ChevronRight size={14} />}
          disabled={page >= totalPages}
          onClick={() => onChange(page + 1)}
        >
          Next
        </Button>
      </div>
    </div>
  );
};