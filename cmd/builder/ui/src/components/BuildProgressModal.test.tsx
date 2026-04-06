import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { BuildProgressModal } from "./BuildProgressModal";

describe("BuildProgressModal", () => {
    const defaultProps = {
        isOpen: true,
        logLines: [] as string[],
        status: "building" as const,
        onClose: vi.fn(),
    };

    it("renders nothing when not open", () => {
        const { container } = render(
            <BuildProgressModal {...defaultProps} isOpen={false} />,
        );
        expect(container.firstChild).toBeNull();
    });

    it('shows "Building…" title when building', () => {
        render(<BuildProgressModal {...defaultProps} />);
        expect(screen.getByText(/Building…/)).toBeInTheDocument();
    });

    it('shows "Build Complete" title on success', () => {
        render(<BuildProgressModal {...defaultProps} status="success" />);
        expect(screen.getByText(/Build Complete/)).toBeInTheDocument();
    });

    it('shows "Build Failed" title on error', () => {
        render(<BuildProgressModal {...defaultProps} status="error" />);
        expect(screen.getByText(/Build Failed/)).toBeInTheDocument();
    });

    it("renders log lines", () => {
        render(
            <BuildProgressModal
                {...defaultProps}
                logLines={["Compiling main.go", "Linking binary"]}
            />,
        );
        expect(screen.getByText("Compiling main.go")).toBeInTheDocument();
        expect(screen.getByText("Linking binary")).toBeInTheDocument();
    });

    it("shows error message on error status", () => {
        render(
            <BuildProgressModal
                {...defaultProps}
                status="error"
                errorMessage="compiler failed"
            />,
        );
        expect(screen.getByText("compiler failed")).toBeInTheDocument();
    });

    it("shows build result details on success", () => {
        render(
            <BuildProgressModal
                {...defaultProps}
                status="success"
                result={{
                    binary_path: "/output/safeguard",
                    size: 10485760,
                    checksum: "abc123",
                }}
            />,
        );
        expect(screen.getByText("safeguard")).toBeInTheDocument();
        expect(screen.getByText("10.00 MB")).toBeInTheDocument();
        expect(screen.getByText("abc123")).toBeInTheDocument();
    });

    it("shows download link on success with result", () => {
        render(
            <BuildProgressModal
                {...defaultProps}
                status="success"
                result={{
                    binary_path: "/output/safeguard",
                    size: 1024,
                    checksum: "abc",
                }}
            />,
        );
        expect(screen.getByText("Download Now")).toBeInTheDocument();
    });

    it("does not show close button while building", () => {
        render(<BuildProgressModal {...defaultProps} status="building" />);
        expect(
            screen.queryByRole("button", { name: "Close" }),
        ).not.toBeInTheDocument();
    });

    it("shows close button when build is complete", () => {
        render(<BuildProgressModal {...defaultProps} status="success" />);
        expect(screen.getByText("Close")).toBeInTheDocument();
    });

    it("shows compiling status indicator while building", () => {
        render(<BuildProgressModal {...defaultProps} status="building" />);
        expect(screen.getByText(/Compiling…/)).toBeInTheDocument();
    });
});
