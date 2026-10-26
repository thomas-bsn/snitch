from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.setup_progress import SetupProgress
from ...models.setup_step_input import SetupStepInput
from typing import cast



def _get_kwargs(
    *,
    body: SetupStepInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/admin/setup-progress",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | SetupProgress:
    if response.status_code == 200:
        response_200 = SetupProgress.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | SetupProgress]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: SetupStepInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | SetupProgress]:
    """  Administrateurs et Super-administrateur : note qu'une étape facultative a été traversée (done) ou
    passée (skipped). Renvoie la progression.

    Args:
        origin (str):
        x_csrf_token (str):
        body (SetupStepInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SetupProgress]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    body: SetupStepInput,
    origin: str,
    x_csrf_token: str,

) -> Error | SetupProgress | None:
    """  Administrateurs et Super-administrateur : note qu'une étape facultative a été traversée (done) ou
    passée (skipped). Renvoie la progression.

    Args:
        origin (str):
        x_csrf_token (str):
        body (SetupStepInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SetupProgress
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: SetupStepInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | SetupProgress]:
    """  Administrateurs et Super-administrateur : note qu'une étape facultative a été traversée (done) ou
    passée (skipped). Renvoie la progression.

    Args:
        origin (str):
        x_csrf_token (str):
        body (SetupStepInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SetupProgress]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: SetupStepInput,
    origin: str,
    x_csrf_token: str,

) -> Error | SetupProgress | None:
    """  Administrateurs et Super-administrateur : note qu'une étape facultative a été traversée (done) ou
    passée (skipped). Renvoie la progression.

    Args:
        origin (str):
        x_csrf_token (str):
        body (SetupStepInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SetupProgress
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
