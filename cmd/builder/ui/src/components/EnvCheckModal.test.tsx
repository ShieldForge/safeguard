import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EnvCheckModal } from "./EnvCheckModal";
import type { EnvStatus } from "../types";

describe("EnvCheckModal", () => {
    const defaultProps = {
        onClose: vi.fn(),
        onRecheck: vi.fn(),
    };

    it("shows loading spinner while checking", () => {
        const { container } = render(
            <EnvCheckModal
                {...defaultProps}
                status={null}
                isChecking={true}
            />,
        );
        expect(
            screen.getByText("Running environment checks…"),
        ).toBeInTheDocument();
        expect(
            container.querySelector(".env-checking-spinner"),
        ).toBeInTheDocument();
    });

    it("shows prompt when not checking and no status", () => {
        render(
            <EnvCheckModal
                {...defaultProps}
                status={null}
                isChecking={false}
            />,
        );
        expect(
            screen.getByText(/Click the button below/),
        ).toBeInTheDocument();
    });

    it("shows all-passed summary when env is ready", () => {
        const status: EnvStatus = {
            status: "ready",
            info: ["Go: 1.21.0", "GCC: 12.2.0"],
        };
        render(
            <EnvCheckModal
                {...defaultProps}
                status={status}
                isChecking={false}
            />,
        );
        expect(screen.getByText(/All checks passed/)).toBeInTheDocument();
    });

    it("shows failing checks when env is not ready", () => {
        const status: EnvStatus = {
            status: "not_ready",
            issues: ["Go compiler not found"],
            info: ["GCC: 12.2.0"],
        };
        render(
            <EnvCheckModal
                {...defaultProps}
                status={status}
                isChecking={false}
            />,
        );
        expect(screen.getByText(/of.*checks passed/)).toBeInTheDocument();
        expect(screen.getByText("Go Compiler")).toBeInTheDocument();
    });

    it("calls onClose when Close button is clicked", async () => {
        const user = userEvent.setup();
        const onClose = vi.fn();
        render(
            <EnvCheckModal
                onClose={onClose}
                onRecheck={vi.fn()}
                status={null}
                isChecking={false}
            />,
        );
        await user.click(screen.getByText("Close"));
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("calls onRecheck when Run Check button is clicked", async () => {
        const user = userEvent.setup();
        const onRecheck = vi.fn();
        render(
            <EnvCheckModal
                onClose={vi.fn()}
                onRecheck={onRecheck}
                status={null}
                isChecking={false}
            />,
        );
        await user.click(screen.getByText("Run Check"));
        expect(onRecheck).toHaveBeenCalledOnce();
    });

    it("shows Re-check button when status already exists", () => {
        const status: EnvStatus = { status: "ready", info: ["Go: 1.21.0"] };
        render(
            <EnvCheckModal
                {...defaultProps}
                status={status}
                isChecking={false}
            />,
        );
        expect(screen.getByText("Re-check")).toBeInTheDocument();
    });

    it("disables recheck button while checking", () => {
        render(
            <EnvCheckModal
                {...defaultProps}
                status={null}
                isChecking={true}
            />,
        );
        expect(screen.getByText("Checking…")).toBeDisabled();
    });

    it("closes on Escape key", async () => {
        const user = userEvent.setup();
        const onClose = vi.fn();
        render(
            <EnvCheckModal
                onClose={onClose}
                onRecheck={vi.fn()}
                status={null}
                isChecking={false}
            />,
        );
        await user.keyboard("{Escape}");
        expect(onClose).toHaveBeenCalledOnce();
    });
});
