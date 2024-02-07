---
page_title: "Abac Rules in Raito Cloud"
---

# Abac Rules in the Raito Provider

Access provider resources (grants, masks, filters, and purposes) can define what and/or who items are authorized based on Attribute-Based Access Control (ABAC) rules. These rules are specified in JSON format following the outlined structure below.

## JSON Structure

The JSON structure for ABAC rules consists of nested objects and arrays that represent various logical expressions:

* **AbacRule:**
    * `literal`: (Optional) A boolean value directly representing a truth condition.
    * `comparison`: (Optional) A `Comparison` expression involving an operator, left operand, and right operand.
    * `aggregator`: (Optional) An `Aggregator` expression combining multiple binary expressions using AND or OR operations.
    * `unaryExpression`: (Optional) A single expression negated using a NOT operator.

  Exactly one argument should be defined.

* **Comparison:**
    * `operator`: Indicates the comparison type (e.g., `HasTag`, `ContainsTag`, `PropertyEquals`, `PropertyIn`).
    * `leftOperand`: The tag key used in the comparison.
    * `rightOperand`: The value to compare with as `Operand`.

* **Operand:**
    * `literal`: A `Literal` value, including booleans, strings, or string lists.

* **Literal:**
    * `bool`: (Optional) A boolean value.
    * `string`: (Optional) A string value.
    * `stringList`: (Optional) A list of string values.

  Exactly one argument should be defined.

* **Aggregator:**
    * `operator`: Specifies the aggregation type (e.g., `And` or `Or`).
    * `operands`: An array of `AbacRule` objects.

* **Unary:**
    * `operator`: Specifies the unary type (e.g., `Not`)
    * `operands`: An array of `AbacRule` objects.

## Constraints

The following constraints will be evaluated during the creation of the abac rule

* The first level should be an aggregation with operator `Or`.
* The second level should be an aggregation with operator `And`.
* The third level can be either an `Comparison`, `Literal` or `Unary` expression.
* If an unary expression is used on the third level, the fourth level should be an `Comparison` or a `Literal`.

## Example Rule

Here's an example JSON rule representing a condition that would evaluate true if a tag `department` has the value `Finance` and the tag `sensitivity` has the value `PII`.

```json
{
  "aggregator": {
    "operator": "Or",
    "operands": [
      {
        "aggregator": {
          "operator": "And",
          "operands": [
            {
              "comparison": {
                "operator": "HasTag",
                "leftOperand": "department",
                "rightOperand": {
                  "literal": {
                    "string": "Finance"
                  }
                }
              }
            },
            {
              "comparison": {
                "operator": "HasTag",
                "leftOperand": "sensitivity",
                "rightOperand": {
                  "literal": {
                    "string": "PII"
                  }
                }
              }
            }
          ]
        }
      }
    ]
  }
} 
```

## Example in Terraform

```terraform
locals {
  example_grant_abac_rule = jsonencode(
    {
      aggregator : {
        operator : "Or"
        operands : [
          {
            aggregator : {
              operator : "And",
              operands : [
                {
                  comparison : {
                    operator : "HasTag",
                    leftOperand : "department",
                    rightOperand : {
                      literal : {
                        string : "Finance"
                      }
                    }
                  }
                },
                {
                  comparison : {
                    operator : "HasTag",
                    leftOperand : "sensitivity",
                    rightOperand : {
                      string : "PII"
                    }
                  }
                }
              ]
            }
          }
        ]
      }
    }
  )
}

resource "raito_datasource" "ds" {
  name = "exampleDS"
}

resource "raito_grant" "example_grant" {
  name        = "Grant with abac"
  description = "Grant with what abac rule"
  state       = "Active"
  what_abac_rule = {
    rule = local.example_grant_abac_rule
  }
  data_source = raito_datasource.ds.id
}
```
