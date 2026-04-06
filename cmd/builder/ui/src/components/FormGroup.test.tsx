import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { FormGroup } from "./FormGroup";

describe("FormGroup", () => {
    it("renders the label text", () => {
        render(
            <FormGroup label="Username:">
                <input />
            </FormGroup>,
        );
        expect(screen.getByText("Username:")).toBeInTheDocument();
    });

    it("shows required badge when importance is required", () => {
        render(
            <FormGroup label="Email:" importance="required">
                <input />
            </FormGroup>,
        );
        expect(screen.getByText("required")).toBeInTheDocument();
    });

    it("shows important badge when importance is important", () => {
        render(
            <FormGroup label="Name:" importance="important">
                <input />
            </FormGroup>,
        );
        expect(screen.getByText("important")).toBeInTheDocument();
    });

    it("does not show badges when no importance specified", () => {
        render(
            <FormGroup label="Optional:">
                <input />
            </FormGroup>,
        );
        expect(screen.queryByText("required")).not.toBeInTheDocument();
        expect(screen.queryByText("important")).not.toBeInTheDocument();
    });

    it("renders tooltip when tooltip prop provided", () => {
        render(
            <FormGroup label="Field:" tooltip="Some help">
                <input />
            </FormGroup>,
        );
        expect(screen.getByText("?")).toBeInTheDocument();
    });

    it("renders children", () => {
        render(
            <FormGroup label="Field:">
                <input data-testid="child-input" />
            </FormGroup>,
        );
        expect(screen.getByTestId("child-input")).toBeInTheDocument();
    });
});
