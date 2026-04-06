import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EnvironmentCheck } from "./EnvironmentCheck";
import type { EnvStatus } from "../types";

describe("EnvironmentCheck", () => {
    it("renders the check button", () => {
        render(
            <EnvironmentCheck onCheck={vi.fn()} status={null} isChecking={false} />,
        );
        expect(
            screen.getByText("Check Build Environment"),
        ).toBeInTheDocument();
    });

    it('shows "Checking..." while checking', () => {
        render(
            <EnvironmentCheck onCheck={vi.fn()} status={null} isChecking={true} />,
        );
        expect(screen.getByText("Checking...")).toBeInTheDocument();
        expect(screen.getByRole("button")).toBeDisabled();
    });

    it("calls onCheck when button is clicked", async () => {
        const user = userEvent.setup();
        const onCheck = vi.fn();
        render(
            <EnvironmentCheck onCheck={onCheck} status={null} isChecking={false} />,
        );
        await user.click(screen.getByText("Check Build Environment"));
        expect(onCheck).toHaveBeenCalledOnce();
    });

    it("shows success message when status is ready", () => {
        const status: EnvStatus = { status: "ready" };
        render(
            <EnvironmentCheck onCheck={vi.fn()} status={status} isChecking={false} />,
        );
        expect(
            screen.getByText(/Build environment is ready/),
        ).toBeInTheDocument();
    });

    it("shows issues when status is not_ready", () => {
        const status: EnvStatus = {
            status: "not_ready",
            issues: ["Go compiler not found", "GCC not found"],
        };
        render(
            <EnvironmentCheck onCheck={vi.fn()} status={status} isChecking={false} />,
        );
        expect(screen.getByText("Go compiler not found")).toBeInTheDocument();
        expect(screen.getByText("GCC not found")).toBeInTheDocument();
    });

    it("shows error message when status is error", () => {
        const status: EnvStatus = {
            status: "error",
            issues: ["Network error"],
        };
        render(
            <EnvironmentCheck onCheck={vi.fn()} status={status} isChecking={false} />,
        );
        expect(screen.getByText(/Network error/)).toBeInTheDocument();
    });

    it("shows environment info when provided", () => {
        const status: EnvStatus = {
            status: "ready",
            info: ["Go: 1.21.0", "GCC: 12.2.0"],
        };
        render(
            <EnvironmentCheck onCheck={vi.fn()} status={status} isChecking={false} />,
        );
        expect(screen.getByText("Go: 1.21.0")).toBeInTheDocument();
        expect(screen.getByText("GCC: 12.2.0")).toBeInTheDocument();
    });
});
