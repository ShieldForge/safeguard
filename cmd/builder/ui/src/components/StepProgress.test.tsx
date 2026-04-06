import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StepProgress } from "./StepProgress";
import type { Step } from "../stores/steps";

const baseSteps: Step[] = [
    { key: "env", label: "Check Environment", status: "complete" },
    { key: "connection", label: "Connection & Auth", status: "current" },
    { key: "mount", label: "Mount & Paths", status: "upcoming" },
];

describe("StepProgress", () => {
    it("renders all step labels", () => {
        render(<StepProgress steps={baseSteps} />);
        expect(screen.getByText("Check Environment")).toBeInTheDocument();
        expect(screen.getByText("Connection & Auth")).toBeInTheDocument();
        expect(screen.getByText("Mount & Paths")).toBeInTheDocument();
    });

    it("shows checkmark for complete steps", () => {
        render(<StepProgress steps={baseSteps} />);
        expect(screen.getByText("✓")).toBeInTheDocument();
    });

    it("shows step numbers for non-complete steps", () => {
        render(<StepProgress steps={baseSteps} />);
        expect(screen.getByText("2")).toBeInTheDocument();
        expect(screen.getByText("3")).toBeInTheDocument();
    });

    it("shows spinner for env step when checkingEnv is true", () => {
        const steps: Step[] = [
            { key: "env", label: "Check Environment", status: "current" },
        ];
        const { container } = render(
            <StepProgress steps={steps} checkingEnv={true} />,
        );
        expect(container.querySelector(".step-spinner")).toBeInTheDocument();
    });

    it("renders 'Check Environment' for env step when not complete", () => {
        const steps: Step[] = [
            { key: "env", label: "Check Environment", status: "current" },
        ];
        render(<StepProgress steps={steps} />);
        expect(screen.getByText("Check Environment")).toBeInTheDocument();
    });

    it("makes env step clickable", () => {
        const steps: Step[] = [
            { key: "env", label: "Check Environment", status: "current" },
        ];
        render(<StepProgress steps={steps} />);
        const envItem = screen.getByRole("button");
        expect(envItem).toBeInTheDocument();
    });
});
