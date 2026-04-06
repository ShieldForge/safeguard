import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CheckEnvSection } from "./CheckEnvSection";
import { makeFormData } from "../../test/helpers";
import type { EnvStatus } from "../../types";

describe("CheckEnvSection", () => {
    const defaultProps = {
        envStatus: null as EnvStatus | null,
        isChecking: false,
        formData: makeFormData(),
        onChange: vi.fn(),
        onCheck: vi.fn(),
    };

    it("renders section title", () => {
        render(<CheckEnvSection {...defaultProps} />);
        expect(screen.getByText("Check Environment")).toBeInTheDocument();
    });

    it("renders prerequisite list", () => {
        render(<CheckEnvSection {...defaultProps} />);
        expect(screen.getByText(/Go compiler/)).toBeInTheDocument();
        expect(screen.getByText(/GCC.*CGO toolchain/)).toBeInTheDocument();
    });

    it("renders source/work/output directory fields", () => {
        render(<CheckEnvSection {...defaultProps} />);
        expect(
            screen.getByPlaceholderText("Auto-detected"),
        ).toBeInTheDocument();
    });

    it("shows check button", () => {
        render(<CheckEnvSection {...defaultProps} />);
        expect(
            screen.getByText("Check Build Environment"),
        ).toBeInTheDocument();
    });

    it("calls onCheck when button is clicked", async () => {
        const user = userEvent.setup();
        const onCheck = vi.fn();
        render(<CheckEnvSection {...defaultProps} onCheck={onCheck} />);
        await user.click(screen.getByText("Check Build Environment"));
        expect(onCheck).toHaveBeenCalledOnce();
    });

    it("shows ready badge when environment is ready", () => {
        render(
            <CheckEnvSection
                {...defaultProps}
                envStatus={{ status: "ready" }}
            />,
        );
        expect(screen.getByText("Environment ready")).toBeInTheDocument();
    });

    it("shows Checking… while checking", () => {
        render(<CheckEnvSection {...defaultProps} isChecking={true} />);
        expect(screen.getByText("Checking…")).toBeInTheDocument();
    });
});
