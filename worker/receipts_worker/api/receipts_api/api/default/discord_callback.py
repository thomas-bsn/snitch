from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    state: str,
    code: str | Unset = UNSET,
    error: str | Unset = UNSET,
    accept: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(accept, Unset):
        headers["Accept"] = accept





    params: dict[str, Any] = {}

    params["state"] = state

    params["code"] = code

    params["error"] = error


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/auth/discord/callback",
        "params": params,
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error:
    if response.status_code == 303:
        response_303 = cast(Any, None)
        return response_303

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())



        return response_400

    if response.status_code == 403:
        response_403 = Error.from_dict(response.json())



        return response_403

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())



        return response_409

    if response.status_code == 429:
        response_429 = Error.from_dict(response.json())



        return response_429

    if response.status_code == 502:
        response_502 = Error.from_dict(response.json())



        return response_502

    if response.status_code == 503:
        response_503 = Error.from_dict(response.json())



        return response_503

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    state: str,
    code: str | Unset = UNSET,
    error: str | Unset = UNSET,
    accept: str | Unset = UNSET,

) -> Response[Any | Error]:
    """  Consumes a single-use state bound to the browser (10 minutes). A link must return in the initiating,
    still-valid session. Supply either code or error, never both. Provider tokens stay on the server and
    are discarded after reading identity. Browser HTML requests return to the SPA with a notice on
    failure.

    Args:
        state (str):
        code (str | Unset):
        error (str | Unset):
        accept (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        state=state,
code=code,
error=error,
accept=accept,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    state: str,
    code: str | Unset = UNSET,
    error: str | Unset = UNSET,
    accept: str | Unset = UNSET,

) -> Any | Error | None:
    """  Consumes a single-use state bound to the browser (10 minutes). A link must return in the initiating,
    still-valid session. Supply either code or error, never both. Provider tokens stay on the server and
    are discarded after reading identity. Browser HTML requests return to the SPA with a notice on
    failure.

    Args:
        state (str):
        code (str | Unset):
        error (str | Unset):
        accept (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return sync_detailed(
        client=client,
state=state,
code=code,
error=error,
accept=accept,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    state: str,
    code: str | Unset = UNSET,
    error: str | Unset = UNSET,
    accept: str | Unset = UNSET,

) -> Response[Any | Error]:
    """  Consumes a single-use state bound to the browser (10 minutes). A link must return in the initiating,
    still-valid session. Supply either code or error, never both. Provider tokens stay on the server and
    are discarded after reading identity. Browser HTML requests return to the SPA with a notice on
    failure.

    Args:
        state (str):
        code (str | Unset):
        error (str | Unset):
        accept (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        state=state,
code=code,
error=error,
accept=accept,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    state: str,
    code: str | Unset = UNSET,
    error: str | Unset = UNSET,
    accept: str | Unset = UNSET,

) -> Any | Error | None:
    """  Consumes a single-use state bound to the browser (10 minutes). A link must return in the initiating,
    still-valid session. Supply either code or error, never both. Provider tokens stay on the server and
    are discarded after reading identity. Browser HTML requests return to the SPA with a notice on
    failure.

    Args:
        state (str):
        code (str | Unset):
        error (str | Unset):
        accept (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return (await asyncio_detailed(
        client=client,
state=state,
code=code,
error=error,
accept=accept,

    )).parsed
