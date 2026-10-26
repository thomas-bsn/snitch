from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.attribution import Attribution
from ...models.attribution_conflict import AttributionConflict
from ...models.attribution_input import AttributionInput
from ...models.error import Error
from typing import cast



def _get_kwargs(
    id: int,
    *,
    body: AttributionInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/local-speakers/{id}/correction".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Attribution | AttributionConflict | Error:
    if response.status_code == 200:
        response_200 = Attribution.from_dict(response.json())



        return response_200

    if response.status_code == 409:
        response_409 = AttributionConflict.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Attribution | AttributionConflict | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    body: AttributionInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Attribution | AttributionConflict | Error]:
    """  Utilisateurs connectés, pour un clip visible. expected_person_id (Personne vue par l'utilisateur)
    est obligatoire pour qu'une correction concurrente n'en écrase pas une autre.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (AttributionInput): Exactement un de person_id ou new_person_name.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Attribution | AttributionConflict | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient,
    body: AttributionInput,
    origin: str,
    x_csrf_token: str,

) -> Attribution | AttributionConflict | Error | None:
    """  Utilisateurs connectés, pour un clip visible. expected_person_id (Personne vue par l'utilisateur)
    est obligatoire pour qu'une correction concurrente n'en écrase pas une autre.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (AttributionInput): Exactement un de person_id ou new_person_name.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Attribution | AttributionConflict | Error
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    body: AttributionInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Attribution | AttributionConflict | Error]:
    """  Utilisateurs connectés, pour un clip visible. expected_person_id (Personne vue par l'utilisateur)
    est obligatoire pour qu'une correction concurrente n'en écrase pas une autre.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (AttributionInput): Exactement un de person_id ou new_person_name.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Attribution | AttributionConflict | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient,
    body: AttributionInput,
    origin: str,
    x_csrf_token: str,

) -> Attribution | AttributionConflict | Error | None:
    """  Utilisateurs connectés, pour un clip visible. expected_person_id (Personne vue par l'utilisateur)
    est obligatoire pour qu'une correction concurrente n'en écrase pas une autre.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (AttributionInput): Exactement un de person_id ou new_person_name.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Attribution | AttributionConflict | Error
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
